package chat

import (
	"cmp"
	"fmt"
	"github.com/ayn2op/tview/layout"
	"log/slog"
	"time"

	"github.com/ayn2op/arikawa/v3/discord"
	"github.com/ayn2op/arikawa/v3/gateway"
	"github.com/ayn2op/arikawa/v3/session"
	"github.com/ayn2op/arikawa/v3/state"
	"github.com/ayn2op/arikawa/v3/state/store/defaultstore"
	"github.com/ayn2op/arikawa/v3/utils/handler"
	"github.com/ayn2op/arikawa/v3/utils/httputil"
	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/discordo/internal/consts"
	clientgateway "github.com/ayn2op/discordo/internal/gateway"
	"github.com/ayn2op/discordo/internal/http"
	"github.com/ayn2op/discordo/internal/ui"
	"github.com/ayn2op/discordo/internal/ui/chat/attachmentspicker"
	"github.com/ayn2op/discordo/internal/ui/chat/channelspicker"
	"github.com/ayn2op/discordo/internal/ui/chat/composer"
	"github.com/ayn2op/discordo/internal/ui/chat/guildstree"
	"github.com/ayn2op/discordo/internal/ui/chat/memberstree"
	"github.com/ayn2op/discordo/internal/ui/chat/mentionslist"
	"github.com/ayn2op/discordo/internal/ui/chat/messageslist"
	"github.com/ayn2op/ningen/v3"
	"github.com/ayn2op/tview"
	"github.com/ayn2op/tview/backdrop"
	"github.com/ayn2op/tview/center"
	"github.com/ayn2op/tview/column"
	"github.com/ayn2op/tview/inert"
	"github.com/ayn2op/tview/keybind"
	"github.com/ayn2op/tview/picker"
	"github.com/ayn2op/tview/row"
	"github.com/ayn2op/tview/stack"
)

type Model struct {
	// focused is the pane that receives keys.
	focused pane
	// guildsTreeVisible reports whether the guilds tree is shown left of the messages.
	guildsTreeVisible bool
	// membersTreeVisible reports whether the members tree is shown right of the messages when the selected channel has members.
	membersTreeVisible bool
	// overlay is the picker shown on top, which takes all input.
	overlay overlay

	guildsTree     guildstree.Model
	membersTree    memberstree.Model
	messagesList   messageslist.Model
	composer       composer.Model
	channelsPicker channelspicker.Model
	// attachmentsPicker picks an attachment or link of the selected message.
	attachmentsPicker attachmentspicker.Model

	selectedChannel *discord.Channel
	// windowUnfocused reports whether the terminal window lost focus.
	windowUnfocused bool

	state  *ningen.State
	events chan gateway.Event

	// typers maps the users typing in the selected channel to when their typing indicator expires.
	typers map[discord.UserID]time.Time

	cfg *config.Config
}

var _ tview.Model[Model] = Model{}

func NewModel(cfg *config.Config, token string) Model {
	m := Model{
		typers: make(map[discord.UserID]time.Time),

		cfg: cfg,
	}

	id := gateway.NewIdentifier(gateway.IdentifyCommand{
		Token:      token,
		Properties: http.IdentifyProperties(),
		Capabilities: gateway.LazyUserNotes |
			gateway.VersionedReadStates |
			gateway.VersionedUserGuildSetttings |
			gateway.DedupeUserObjects |
			gateway.PrioritizedReadyPayload |
			gateway.MultipleGuildExperimentPopulations |
			gateway.NonChannelReadStates |
			gateway.AuthTokenRefresh |
			gateway.DebounceMessageReactions,
	})

	session := session.NewWithGateway(clientgateway.New(id), handler.New())
	session.Client = http.NewClient(token)
	state := state.NewFromSession(session, defaultstore.New())
	m.state = ningen.FromState(state)

	m.events = make(chan gateway.Event)
	m.state.AddChanHandler(m.events)
	m.state.StateLog = func(err error) {
		slog.Error("state log", "err", err)
	}
	m.state.OnRequest = append(m.state.OnRequest, httputil.WithHeaders(http.Headers()), onRequest)

	m.guildsTree = guildstree.NewModel(cfg, m.state)
	m.membersTree = memberstree.NewModel(cfg, m.state)
	m.messagesList = messageslist.NewModel(cfg, m.state)
	m.composer = composer.NewModel(cfg, m.state)
	m.channelsPicker = channelspicker.NewModel(cfg)
	m.attachmentsPicker = attachmentspicker.NewModel(cfg)

	m.guildsTreeVisible = cfg.Sidebar.Visible
	m.membersTreeVisible = cfg.MembersTree.Visible
	if m.guildsTreeVisible {
		// The guilds tree is focused first at start-up when visible.
		m.focused = paneGuildsTree
	} else {
		m.focused = paneMessagesList
	}
	return m
}

// overlay is the picker shown on top of the panes, if any.
type overlay int

const (
	overlayNone overlay = iota
	overlayChannelsPicker
	overlayAttachmentsPicker
)

// pickerView returns the view of the picker shown on top, or nil if none is open.
func (m Model) pickerView() tview.Widget {
	switch m.overlay {
	case overlayChannelsPicker:
		return m.channelsPicker.View()
	case overlayAttachmentsPicker:
		return m.attachmentsPicker.View()
	default:
		return nil
	}
}

// togglePicker opens the channels picker, or closes it if it is open.
func (m *Model) togglePicker() {
	if m.overlay == overlayChannelsPicker {
		m.closePicker()
	} else {
		m.openPicker()
	}
}

func (m *Model) openPicker() {
	m.overlay = overlayChannelsPicker
	m.channelsPicker.RefreshChannels(m.state)
}

func (m *Model) closePicker() {
	m.overlay = overlayNone
	m.channelsPicker.Reset()
}

// toggle shows or hides the side pane p and focuses it when shown.
func (m *Model) toggle(visible *bool, p pane) {
	*visible = !*visible
	if *visible {
		m.setFocus(p)
	} else if m.focused == p {
		m.focused = paneMessagesList
	}
}

// membersTreeShown reports whether the members tree is visible and the selected channel has members to show.
func (m Model) membersTreeShown() bool {
	return m.membersTreeVisible && m.membersTree.Shown()
}

// cycleFocus focuses the next pane that can take the focus, going backwards if step is -1.
func (m *Model) cycleFocus(step pane) {
	p := m.focused
	for range paneCount - 1 {
		p = (p + step + paneCount) % paneCount
		if m.canFocus(p) {
			m.focused = p
			return
		}
	}
}

func (m Model) Init() tview.Cmd {
	return tview.Batch(openState(m.state), listen(m.events), m.guildsTree.ListenVoice(), tview.RequestTerminalInfo())
}

// Update returns m changed in response to msg and a command to run, or nil.
func (m Model) Update(msg tview.Msg) (Model, tview.Cmd) {
	switch msg := msg.(type) {
	case gateway.Event:
		return m, tview.Batch(m.applyEvent(msg), listen(m.events))
	case guildstree.VoiceMsg:
		return m, tview.Batch(m.updatePane(paneGuildsTree, msg), m.guildsTree.ListenVoice())
	case tview.FocusMsg:
		m.windowUnfocused = !msg.Focused
		return m, nil
	case guildstree.ChannelLoadedMsg:
		node := m.guildsTree.CurrentNode()
		if node == nil {
			return m, nil
		}
		channelID, ok := node.Reference().(discord.ChannelID)
		if !ok || channelID != msg.Channel.ID {
			return m, nil
		}

		membersCmd := m.setSelectedChannel(&msg.Channel)
		clear(m.typers)

		if m.cfg.AutoFocus {
			m.setFocus(paneComposer)
		}
		title := ui.ChannelToString(msg.Channel, m.cfg.Icons, m.state) + " - " + consts.Name
		return m, tview.Batch(tview.SetTitle(title), m.messagesList.SetChannel(&msg.Channel, msg.Messages), membersCmd)
	case messageslist.Msg, tview.TerminalInfoMsg, tview.ResizeMsg:
		return m, m.updatePane(paneMessagesList, msg)
	case typingExpiredMsg:
		if until, ok := m.typers[msg.userID]; ok && !time.Now().Before(until) {
			delete(m.typers, msg.userID)
		}
		return m, nil
	case channelspicker.SelectedMsg:
		return m, m.navigateToChannel(msg.ChannelID)
	case channelspicker.CancelMsg:
		m.closePicker()
		return m, nil
	case attachmentspicker.SelectedMsg:
		m.overlay = overlayNone
		return m, msg.Action
	case attachmentspicker.CancelMsg:
		m.overlay = overlayNone
		return m, nil
	case messageslist.ShowAttachmentsMsg:
		m.attachmentsPicker.SetItems(picker.Items(msg))
		m.overlay = overlayAttachmentsPicker
		return m, nil
	case messageslist.ReplyMsg:
		m.composer.StartReply(msg.Message, msg.Name, msg.Mention)
		m.setFocus(paneComposer)
		return m, nil
	case messageslist.EditMsg:
		m.composer.StartEdit(discord.Message(msg))
		m.setFocus(paneComposer)
		return m, nil
	case composer.EditLastMsg:
		if message, ok := m.messagesList.SelectLastOwn(); ok {
			m.composer.StartEdit(message)
		}
		return m, nil
	case composer.SentMsg:
		m.messagesList.ShowNewest()
		return m, nil
	case QuitMsg:
		return m, closeState(m.state)
	case tview.KeyMsg:
		switch {
		case keybind.Matches(msg, m.cfg.Keybinds.FocusGuildsTree.Keybind):
			m.setFocus(paneGuildsTree)
			return m, nil
		case keybind.Matches(msg, m.cfg.Keybinds.FocusMembersTree.Keybind):
			m.setFocus(paneMembersTree)
			return m, nil
		case keybind.Matches(msg, m.cfg.Keybinds.FocusMessagesList.Keybind):
			m.setFocus(paneMessagesList)
			return m, nil
		case keybind.Matches(msg, m.cfg.Keybinds.FocusComposer.Keybind):
			m.setFocus(paneComposer)
			return m, nil

		case keybind.Matches(msg, m.cfg.Keybinds.FocusPrevious.Keybind):
			m.cycleFocus(-1)
			return m, nil
		case keybind.Matches(msg, m.cfg.Keybinds.FocusNext.Keybind):
			m.cycleFocus(1)
			return m, nil

		case keybind.Matches(msg, m.cfg.Keybinds.ToggleGuildsTree.Keybind):
			m.toggle(&m.guildsTreeVisible, paneGuildsTree)
			return m, nil
		case keybind.Matches(msg, m.cfg.Keybinds.ToggleMembersTree.Keybind):
			m.toggle(&m.membersTreeVisible, paneMembersTree)
			return m, nil
		case keybind.Matches(msg, m.cfg.Keybinds.ToggleChannelsPicker.Keybind):
			m.togglePicker()
			return m, nil
		case keybind.Matches(msg, m.cfg.Keybinds.ToggleMute.Keybind):
			return m, m.guildsTree.ToggleMute()

		case keybind.Matches(msg, m.cfg.Keybinds.Logout.Keybind):
			return m, tview.Sequence(closeState(m.state), logout())
		}
	case composer.TabSuggestMsg, mentionslist.Msg:
		return m, m.updatePane(paneComposer, msg)
	case paneMsg:
		if msg.focus {
			m.setFocus(msg.pane)
		}
		if msg.msg == nil {
			return m, nil
		}
		return m, m.updatePane(msg.pane, msg.msg)
	}
	return m, m.route(msg)
}

// route sends msg to the open picker, which takes all input, and otherwise to the focused pane.
func (m *Model) route(msg tview.Msg) tview.Cmd {
	var cmd tview.Cmd
	switch m.overlay {
	case overlayChannelsPicker:
		m.channelsPicker, cmd = m.channelsPicker.Update(msg)
	case overlayAttachmentsPicker:
		m.attachmentsPicker, cmd = m.attachmentsPicker.Update(msg)
	default:
		cmd = m.updatePane(m.focused, msg)
	}
	return cmd
}

// updatePane updates pane p with msg.
func (m *Model) updatePane(p pane, msg tview.Msg) tview.Cmd {
	var cmd tview.Cmd
	switch p {
	case paneGuildsTree:
		m.guildsTree, cmd = m.guildsTree.Update(msg)
	case paneMessagesList:
		m.messagesList, cmd = m.messagesList.Update(msg)
	case paneComposer:
		m.composer, cmd = m.composer.Update(msg)
	case paneMembersTree:
		m.membersTree, cmd = m.membersTree.Update(msg)
	}
	return cmd
}

// paneWidget passes input to the widget of a pane and marks mouse input within the pane as meant for it.
type paneWidget struct {
	pane  pane
	child tview.Widget
}

func (p paneWidget) Size() (width, height layout.Length) {
	return p.child.Size()
}

func (p paneWidget) Layout(limits layout.Limits) layout.Size {
	return p.child.Layout(limits)
}

func (p paneWidget) Draw(screen tview.Screen, area tview.Rectangle) {
	p.child.Draw(screen, area)
}

func (p paneWidget) Handle(msg tview.Msg, area tview.Rectangle) tview.Msg {
	out := p.child.Handle(msg, area)
	if mouse, ok := msg.(tview.MouseMsg); ok && area.Contains(mouse.Position()) {
		return paneMsg{pane: p.pane, msg: out, focus: mouse.Action == tview.MouseLeftDown}
	}
	// Keys and paste reach only the focused pane, so what it makes of them goes to that pane through route.
	return out
}

// pane is one of the models shown side by side, one of which has the focus.
type pane int

const (
	paneGuildsTree pane = iota
	paneMessagesList
	paneComposer
	paneMembersTree
	paneCount
)

// paneView returns the view of p, focused if it has the focus and no picker is open, marked so that clicking it focuses it.
func (m Model) paneView(p pane) tview.Widget {
	focused := m.focused == p && m.overlay == overlayNone
	var child tview.Widget
	switch p {
	case paneGuildsTree:
		child = m.guildsTree.View(focused)
	case paneMessagesList:
		child = m.messagesList.View(focused, m.typingFooter())
	case paneComposer:
		child = m.composer.View(focused)
	case paneMembersTree:
		child = m.membersTree.View(focused)
	}
	return paneWidget{pane: p, child: child}
}

// setFocus focuses p if it can take the focus.
func (m *Model) setFocus(p pane) {
	if m.canFocus(p) {
		m.focused = p
	}
}

// canFocus reports whether p is shown and can take the focus.
func (m Model) canFocus(p pane) bool {
	switch p {
	case paneGuildsTree:
		return m.guildsTreeVisible
	case paneComposer:
		return !m.composer.Disabled()
	case paneMembersTree:
		return m.membersTreeShown()
	default:
		return true
	}
}

// View shows the guilds tree left of the messages above the composer, with the mentions list over the messages and an open picker on top.
func (m Model) View() tview.Widget {
	picker := m.pickerView()

	var right tview.Widget = column.New(
		m.paneView(paneMessagesList),
		column.New(m.paneView(paneComposer)).Height(layout.Fixed(m.composer.Height())),
	)
	if mentions := m.composer.MentionsListView(m.focused == paneComposer); mentions != nil {
		right = stack.New(right, mentions)
	}

	main := right
	if m.membersTreeShown() {
		width := m.cfg.MembersTree.WidthPercent
		main = row.New(
			column.New(main).Width(layout.FillPortion(100-width)),
			column.New(m.paneView(paneMembersTree)).Width(layout.FillPortion(width)),
		)
	}
	if m.guildsTreeVisible {
		width := m.cfg.Sidebar.WidthPercent
		main = row.New(
			column.New(m.paneView(paneGuildsTree)).Width(layout.FillPortion(width)),
			column.New(main).Width(layout.FillPortion(100-width)),
		)
	}

	if picker == nil {
		return main
	}
	return stack.New(
		// The panes behind the picker take no input.
		inert.New(main),
		backdrop.New().Style(m.cfg.Theme.Dialog.BackgroundStyle.Style),
		center.New(column.New(picker).Width(layout.Fixed(m.cfg.Picker.Width)).Height(layout.Fixed(m.cfg.Picker.Height))),
	)
}

// typingFooter returns who is typing in the selected channel, or "" if nobody is.
func (m Model) typingFooter() string {
	channel := m.selectedChannel
	if channel == nil || len(m.typers) == 0 {
		return ""
	}
	var names []string
	for userID := range m.typers {
		var name string
		if channel.GuildID.IsValid() {
			member, err := m.state.Cabinet.Member(channel.GuildID, userID)
			if err != nil {
				continue
			}
			name = cmp.Or(member.Nick, member.User.DisplayName, member.User.Username)
		} else {
			for _, recipient := range channel.DMRecipients {
				if recipient.ID == userID {
					name = recipient.DisplayOrUsername()
					break
				}
			}
		}
		if name != "" {
			names = append(names, name)
		}
	}

	switch len(names) {
	case 0:
		return ""
	case 1:
		return names[0] + " is typing..."
	case 2:
		return fmt.Sprintf("%s and %s are typing...", names[0], names[1])
	case 3:
		return fmt.Sprintf("%s, %s, and %s are typing...", names[0], names[1], names[2])
	default:
		return "Several people are typing..."
	}
}
