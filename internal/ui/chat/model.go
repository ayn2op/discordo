package chat

import (
	"fmt"
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
	// channelsPickerOpen and attachmentsPickerOpen report whether that picker is shown on top and takes all input.
	channelsPickerOpen    bool
	attachmentsPickerOpen bool

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
		m.focused = guildsTreePane
	} else {
		m.focused = messagesListPane
	}
	return m
}

func (m *Model) setSelectedChannel(channel *discord.Channel) tview.Cmd {
	m.selectedChannel = channel
	m.composer.SetChannel(channel)
	cmd := m.membersTree.SetChannel(channel)
	if !m.canFocus(m.focused) {
		m.focused = messagesListPane
	}
	return cmd
}

func (m *Model) togglePicker() tview.Cmd {
	if m.channelsPickerOpen {
		return m.closePicker()
	}
	return m.openPicker()
}

func (m *Model) openPicker() tview.Cmd {
	m.channelsPickerOpen = true
	m.channelsPicker.RefreshChannels(m.state)
	return nil
}

func (m *Model) closePicker() tview.Cmd {
	m.channelsPickerOpen = false
	m.channelsPicker.Reset()
	return nil
}

func (m *Model) closeAttachmentsPicker() tview.Cmd {
	m.attachmentsPickerOpen = false
	return nil
}

// pickerOpen reports whether a picker is shown on top.
func (m Model) pickerOpen() bool {
	return m.channelsPickerOpen || m.attachmentsPickerOpen
}

// pickerView returns the view of the picker shown on top, or nil if none is open.
func (m Model) pickerView() tview.Element {
	switch {
	case m.channelsPickerOpen:
		return m.channelsPicker.View()
	case m.attachmentsPickerOpen:
		return m.attachmentsPicker.View()
	default:
		return nil
	}
}

func (m *Model) navigateToChannel(channelID discord.ChannelID) tview.Cmd {
	closeCmd := m.closePicker()
	var cmd tview.Cmd
	m.guildsTree, cmd = m.guildsTree.Update(guildstree.NavigateMsg{ChannelID: channelID})
	return tview.Sequence(closeCmd, cmd)
}

// toggle shows or hides the side pane p and focuses it when shown.
func (m *Model) toggle(visible *bool, p pane) {
	*visible = !*visible
	if *visible {
		m.setFocus(p)
	} else if m.focused == p {
		m.focused = messagesListPane
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
	return tview.Batch(openState(m.state), listen(m.events), tview.RequestTerminalInfo())
}

func (m Model) Update(msg tview.Msg) (Model, tview.Cmd) {
	cmd := m.update(msg)
	return m, cmd
}

// update changes m in response to msg and returns a command to run, or nil.
func (m *Model) update(msg tview.Msg) tview.Cmd {
	switch msg := msg.(type) {
	case gateway.Event:
		return tview.Batch(m.applyEvent(msg), listen(m.events))
	case tview.FocusMsg:
		m.windowUnfocused = !msg.Focused
		return nil
	case guildstree.ChannelLoadedMsg:
		node := m.guildsTree.CurrentNode()
		if node == nil {
			return nil
		}
		channelID, ok := node.Reference().(discord.ChannelID)
		if !ok || channelID != msg.Channel.ID {
			return nil
		}

		membersCmd := m.setSelectedChannel(&msg.Channel)
		m.clearTypers()

		if m.cfg.AutoFocus {
			m.setFocus(composerPane)
		}
		title := ui.ChannelToString(msg.Channel, m.cfg.Icons, m.state) + " - " + consts.Name
		return tview.Batch(tview.SetTitle(title), m.messagesList.SetChannel(&msg.Channel, msg.Messages), membersCmd)
	case messageslist.Msg, tview.TerminalInfoMsg:
		return m.updatePane(messagesListPane, msg)
	case typingExpiredMsg:
		if until, ok := m.typers[msg.userID]; ok && !time.Now().Before(until) {
			m.removeTyper(msg.userID)
		}
		return nil
	case channelspicker.SelectedMsg:
		return m.navigateToChannel(msg.ChannelID)
	case channelspicker.CancelMsg:
		return m.closePicker()
	case attachmentspicker.SelectedMsg:
		return tview.Sequence(msg.Action, m.closeAttachmentsPicker())
	case attachmentspicker.CancelMsg:
		return m.closeAttachmentsPicker()
	case messageslist.ShowAttachmentsMsg:
		m.attachmentsPicker.SetItems(picker.Items(msg))
		m.attachmentsPickerOpen = true
		return nil
	case messageslist.ReplyMsg:
		m.composer.StartReply(msg.Message, msg.Name, msg.Mention)
		m.setFocus(composerPane)
		return nil
	case messageslist.EditMsg:
		m.composer.StartEdit(discord.Message(msg))
		m.setFocus(composerPane)
		return nil
	case composer.EditLastMsg:
		if message, ok := m.messagesList.SelectLastOwn(); ok {
			m.composer.StartEdit(message)
		}
		return nil
	case composer.SentMsg:
		m.messagesList.ShowNewest()
		return nil
	case QuitMsg:
		return closeState(m.state)
	case tview.KeyMsg:
		switch {
		case keybind.Matches(msg, m.cfg.Keybinds.FocusGuildsTree.Keybind):
			m.composer.CloseMentions()
			m.setFocus(guildsTreePane)
			return nil
		case keybind.Matches(msg, m.cfg.Keybinds.FocusMembersTree.Keybind):
			m.composer.CloseMentions()
			m.setFocus(membersTreePane)
			return nil
		case keybind.Matches(msg, m.cfg.Keybinds.FocusMessagesList.Keybind):
			m.composer.CloseMentions()
			m.setFocus(messagesListPane)
			return nil
		case keybind.Matches(msg, m.cfg.Keybinds.FocusComposer.Keybind):
			m.setFocus(composerPane)
			return nil

		case keybind.Matches(msg, m.cfg.Keybinds.FocusPrevious.Keybind):
			m.cycleFocus(-1)
			return nil
		case keybind.Matches(msg, m.cfg.Keybinds.FocusNext.Keybind):
			m.cycleFocus(1)
			return nil

		case keybind.Matches(msg, m.cfg.Keybinds.ToggleGuildsTree.Keybind):
			m.toggle(&m.guildsTreeVisible, guildsTreePane)
			return nil
		case keybind.Matches(msg, m.cfg.Keybinds.ToggleMembersTree.Keybind):
			m.toggle(&m.membersTreeVisible, membersTreePane)
			return nil
		case keybind.Matches(msg, m.cfg.Keybinds.ToggleChannelsPicker.Keybind):
			return m.togglePicker()

		case keybind.Matches(msg, m.cfg.Keybinds.Logout.Keybind):
			return tview.Sequence(closeState(m.state), logout())
		}
	case composer.TabSuggestMsg, mentionslist.Msg:
		return m.updatePane(composerPane, msg)
	case paneMsg:
		if msg.focus {
			m.setFocus(msg.pane)
		}
		if msg.msg == nil {
			return nil
		}
		return m.updatePane(msg.pane, msg.msg)
	}
	return m.route(msg)
}

// route sends msg to the open picker, which takes all input, and otherwise to the focused pane.
func (m *Model) route(msg tview.Msg) tview.Cmd {
	var cmd tview.Cmd
	switch {
	case m.channelsPickerOpen:
		m.channelsPicker, cmd = m.channelsPicker.Update(msg)
	case m.attachmentsPickerOpen:
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
	case guildsTreePane:
		m.guildsTree, cmd = m.guildsTree.Update(msg)
	case messagesListPane:
		m.messagesList, cmd = m.messagesList.Update(msg)
	case composerPane:
		m.composer, cmd = m.composer.Update(msg)
	case membersTreePane:
		m.membersTree, cmd = m.membersTree.Update(msg)
	}
	return cmd
}

// paneMsg is what a pane's element made of a mouse message within it. A left mouse button press also focuses the pane.
type paneMsg struct {
	pane  pane
	msg   tview.Msg
	focus bool
}

// paneElement passes input to the element of a pane and marks mouse input within the pane as meant for it.
type paneElement struct {
	pane  pane
	child tview.Element
}

func (p paneElement) Draw(screen tview.Screen, area tview.Rectangle) {
	p.child.Draw(screen, area)
}

func (p paneElement) Handle(msg tview.Msg, area tview.Rectangle) tview.Msg {
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
	guildsTreePane pane = iota
	messagesListPane
	composerPane
	membersTreePane
	paneCount
)

// paneView returns the view of p, focused if it has the focus and no picker is open, marked so that clicking it focuses it.
func (m Model) paneView(p pane) tview.Element {
	focused := m.focused == p && !m.pickerOpen()
	var child tview.Element
	switch p {
	case guildsTreePane:
		child = m.guildsTree.View(focused)
	case messagesListPane:
		child = m.messagesList.View(focused)
	case composerPane:
		child = m.composer.View(focused)
	case membersTreePane:
		child = m.membersTree.View(focused)
	}
	return paneElement{pane: p, child: child}
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
	case guildsTreePane:
		return m.guildsTreeVisible
	case composerPane:
		return !m.composer.Disabled()
	case membersTreePane:
		return m.membersTreeShown()
	default:
		return true
	}
}

// View shows the guilds tree left of the messages above the composer, with the mentions list over the messages and an open picker on top.
func (m Model) View() tview.Element {
	picker := m.pickerView()

	var right tview.Element = column.New(
		m.paneView(messagesListPane),
		column.New(m.paneView(composerPane)).Height(tview.Fixed(m.composer.Height())),
	)
	if mentions := m.composer.MentionsView(); mentions != nil {
		right = stack.New(right, mentions)
	}

	main := right
	if m.membersTreeShown() {
		width := m.cfg.MembersTree.WidthPercent
		main = row.New(
			column.New(main).Width(tview.FillPortion(100-width)),
			column.New(m.paneView(membersTreePane)).Width(tview.FillPortion(width)),
		)
	}
	if m.guildsTreeVisible {
		width := m.cfg.Sidebar.WidthPercent
		main = row.New(
			column.New(m.paneView(guildsTreePane)).Width(tview.FillPortion(width)),
			column.New(main).Width(tview.FillPortion(100-width)),
		)
	}

	if picker == nil {
		return main
	}
	return stack.New(
		// The panes behind the picker take no input.
		inert.New(main),
		backdrop.New().Style(m.cfg.Theme.Dialog.BackgroundStyle.Style),
		center.New(column.New(picker).Width(tview.Fixed(m.cfg.Picker.Width)).Height(tview.Fixed(m.cfg.Picker.Height))),
	)
}

// typingExpiredMsg ends a user's typing indicator unless they typed again and extended it.
type typingExpiredMsg struct {
	userID discord.UserID
}

func (m *Model) clearTypers() {
	clear(m.typers)
	m.updateFooter()
}

// addTyper shows userID as typing and returns a command that ends it after the typing duration.
func (m *Model) addTyper(userID discord.UserID) tview.Cmd {
	m.typers[userID] = time.Now().Add(composer.TypingDuration)
	m.updateFooter()
	return func() tview.Msg {
		time.Sleep(composer.TypingDuration)
		return typingExpiredMsg{userID}
	}
}

func (m *Model) removeTyper(userID discord.UserID) {
	delete(m.typers, userID)
	m.updateFooter()
}

func (m *Model) updateFooter() {
	selectedChannel := m.selectedChannel
	if selectedChannel == nil {
		return
	}
	guildID := selectedChannel.GuildID

	var footer string
	if len(m.typers) > 0 {
		var names []string
		for userID := range m.typers {
			var name string
			if guildID.IsValid() {
				member, err := m.state.Cabinet.Member(guildID, userID)
				if err != nil {
					slog.Error("failed to get member from state", "err", err, "guild_id", guildID, "user_id", userID)
					continue
				}

				if member.Nick != "" {
					name = member.Nick
				} else {
					name = member.User.DisplayOrUsername()
				}
			} else {
				for _, recipient := range selectedChannel.DMRecipients {
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
		case 1:
			footer = names[0] + " is typing..."
		case 2:
			footer = fmt.Sprintf("%s and %s are typing...", names[0], names[1])
		case 3:
			footer = fmt.Sprintf("%s, %s, and %s are typing...", names[0], names[1], names[2])
		default:
			footer = "Several people are typing..."
		}
	}

	m.messagesList.SetFooter(footer)
}
