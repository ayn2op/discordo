package chat

import (
	"cmp"
	"fmt"
	"github.com/ayn2op/tview/layout"
	"log/slog"
	"slices"
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
	"github.com/ayn2op/discordo/internal/ui/autocomplete"
	"github.com/ayn2op/discordo/internal/ui/chat/attachmentspicker"
	"github.com/ayn2op/discordo/internal/ui/chat/channelspicker"
	"github.com/ayn2op/discordo/internal/ui/chat/composer"
	"github.com/ayn2op/discordo/internal/ui/chat/guildstree"
	"github.com/ayn2op/discordo/internal/ui/chat/memberstree"
	"github.com/ayn2op/discordo/internal/ui/chat/messageslist"
	"github.com/ayn2op/discordo/internal/ui/chat/reactionspicker"
	"github.com/ayn2op/discordo/internal/ui/images"
	"github.com/ayn2op/ningen/v3"
	"github.com/ayn2op/tview"
	"github.com/ayn2op/tview/backdrop"
	"github.com/ayn2op/tview/center"
	"github.com/ayn2op/tview/column"
	tviewimage "github.com/ayn2op/tview/image"
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
	// reactionsPicker picks an emoji to react to the selected message with.
	reactionsPicker reactionspicker.Model

	selectedChannel *discord.Channel
	// windowUnfocused reports whether the terminal window lost focus.
	windowUnfocused bool

	state  *ningen.State
	events chan gateway.Event

	// typers maps the users typing in the selected channel to when their typing indicator expires.
	typers map[discord.UserID]time.Time

	// images are shared by the panes.
	images map[discord.URL]tviewimage.Widget
	kitty  bool
	// kittyImages counts the images sent to the terminal.
	kittyImages int
	// Zero if the terminal does not report its cell size.
	cellWidth, cellHeight uint

	cfg *config.Config
}

func NewModel(cfg *config.Config, token string) Model {
	m := Model{
		typers: make(map[discord.UserID]time.Time),
		images: make(map[discord.URL]tviewimage.Widget),

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
	m.reactionsPicker = reactionspicker.NewModel(cfg, m.state)

	m.guildsTreeVisible = cfg.Sidebar.Visible
	m.membersTreeVisible = cfg.MembersTree.Visible
	if m.guildsTreeVisible {
		// The guilds tree is focused first at start-up when visible.
		m.focused = paneGuildsTree
	} else {
		m.focused = paneMessagesList
	}
	m.syncPanes()
	return m
}

// overlay is the picker shown on top of the panes, if any.
type overlay int

const (
	overlayNone overlay = iota
	overlayChannelsPicker
	overlayAttachmentsPicker
	overlayReactionsPicker
)

// pickerView returns the view of the picker shown on top, or nil if none is open.
func (m Model) pickerView() tview.Widget {
	switch m.overlay {
	case overlayChannelsPicker:
		return m.channelsPicker.View()
	case overlayAttachmentsPicker:
		return m.attachmentsPicker.View()
	case overlayReactionsPicker:
		return m.reactionsPicker.View()
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

// toggle shows or hides the side pane target and focuses it when shown.
func (m *Model) toggle(visible *bool, target pane) {
	*visible = !*visible
	if *visible {
		m.setFocus(target)
	} else if m.focused == target {
		m.focused = paneMessagesList
	}
}

// membersTreeShown reports whether the members tree is visible and the selected channel has members to show.
func (m Model) membersTreeShown() bool {
	return m.membersTreeVisible && m.membersTree.Shown()
}

// cycleFocus focuses the next pane that can take the focus, going backwards if step is -1.
func (m *Model) cycleFocus(step pane) {
	next := m.focused
	for range paneCount - 1 {
		next = (next + step + paneCount) % paneCount
		if m.canFocus(next) {
			m.focused = next
			return
		}
	}
}

// syncPanes passes the panes the state of m that they show.
func (m *Model) syncPanes() {
	m.guildsTree.SetFocused(m.paneFocused(paneGuildsTree))
	m.messagesList.SetFocused(m.paneFocused(paneMessagesList))
	m.composer.SetFocused(m.paneFocused(paneComposer))
	m.membersTree.SetFocused(m.paneFocused(paneMembersTree))
	m.messagesList.SetFooter(m.typingFooter())
	m.messagesList.SetAvatarWidth(int(squareCols(uint(m.cfg.UI.MessagesList.Avatars.Height), m.imageCell())))
}

func (m Model) update(msg tview.Msg) (Model, tview.Cmd) {
	switch msg := msg.(type) {
	case gateway.Event:
		return m, tview.Batch(m.applyEvent(msg), listen(m.events))
	case guildstree.VoiceMsg:
		return m, m.updatePane(paneGuildsTree, msg)
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
		// Not shown again soon.
		return m, tview.Batch(m.clearImages(), tview.SetTitle(title), m.messagesList.SetChannel(&msg.Channel, msg.Messages), membersCmd)
	case messageslist.Msg:
		return m, m.updatePane(paneMessagesList, msg)
	case tview.ResizeMsg:
		cell := m.imageCell()
		cols, rows := msg.Size()
		width, height := msg.PixelSize()
		// A resize without a size in pixels may be followed by the terminal's own report of it.
		if cols > 0 && rows > 0 && width > 0 && height > 0 {
			m.cellWidth, m.cellHeight = uint(width/cols), uint(height/rows)
		}
		return m, m.reloadImages(cell)
	case tview.TerminalInfoMsg:
		cell := m.imageCell()
		m.kitty = m.isKitty(msg.Name)
		return m, m.reloadImages(cell)
	case images.RequestedMsg:
		cmd := m.loadImages(msg)
		return m, tview.Batch(cmd, m.sendImages())
	case images.LoadedMsg:
		// Another channel was selected, or the cell size changed, while it loaded.
		if _, ok := m.images[msg.URL]; !ok || msg.Cell != m.imageCell() {
			return m, nil
		}

		image, cmd := msg.Image, tview.Cmd(nil)
		if m.kitty {
			m.kittyImages++
			image = image.Kitty(m.kittyImages%maxKittyImages + 1).Compressor(newPooledCompressor)
			cmd = image.Transmit()
		}

		m.images[msg.URL] = image
		return m, tview.Batch(cmd, m.sendImages())
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
	case messageslist.ShowReactionsMsg:
		m.overlay = overlayReactionsPicker
		var cmd tview.Cmd
		m.reactionsPicker, cmd = m.reactionsPicker.Update(reactionspicker.ShowMsg(msg))
		return m, cmd
	case reactionspicker.SelectedMsg:
		m.overlay = overlayNone
		return m, m.react(msg)
	case reactionspicker.CancelMsg:
		m.overlay = overlayNone
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
		keybinds := m.cfg.Keybinds
		switch {
		case keybind.Matches(msg, keybinds.FocusGuildsTree.Keybind):
			m.setFocus(paneGuildsTree)
			return m, nil
		case keybind.Matches(msg, keybinds.FocusMembersTree.Keybind):
			m.setFocus(paneMembersTree)
			return m, nil
		case keybind.Matches(msg, keybinds.FocusMessagesList.Keybind):
			m.setFocus(paneMessagesList)
			return m, nil
		case keybind.Matches(msg, keybinds.FocusComposer.Keybind):
			m.setFocus(paneComposer)
			return m, nil

		case keybind.Matches(msg, keybinds.FocusPrevious.Keybind):
			m.cycleFocus(-1)
			return m, nil
		case keybind.Matches(msg, keybinds.FocusNext.Keybind):
			m.cycleFocus(1)
			return m, nil

		case keybind.Matches(msg, keybinds.ToggleGuildsTree.Keybind):
			m.toggle(&m.guildsTreeVisible, paneGuildsTree)
			return m, nil
		case keybind.Matches(msg, keybinds.ToggleMembersTree.Keybind):
			m.toggle(&m.membersTreeVisible, paneMembersTree)
			return m, nil
		case keybind.Matches(msg, keybinds.ToggleChannelsPicker.Keybind):
			m.togglePicker()
			return m, nil
		// The guilds tree is what the voice channel joined is in.
		case keybind.Matches(msg, keybinds.ToggleMute.Keybind, keybinds.ToggleDeafen.Keybind, keybinds.LeaveVoice.Keybind):
			return m, m.updatePane(paneGuildsTree, msg)

		case keybind.Matches(msg, keybinds.Logout.Keybind):
			return m, ui.ShowDialog(
				"Are you sure you want to log out?",
				ui.DialogButton{Label: "No"},
				ui.DialogButton{Label: "Yes", Cmd: tview.Sequence(closeState(m.state), logout())},
			)
		}
	case composer.RefreshAutocompleteMsg, autocomplete.Msg:
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
	case overlayReactionsPicker:
		m.reactionsPicker, cmd = m.reactionsPicker.Update(msg)
	default:
		cmd = m.updatePane(m.focused, msg)
	}
	return cmd
}

// updatePane updates pane target with msg.
func (m *Model) updatePane(target pane, msg tview.Msg) tview.Cmd {
	var cmd tview.Cmd
	switch target {
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
	// Mouse input the pane leaves, but for a press, goes on to the other panes, for a slider dragged out of one to follow the pointer.
	if mouse, ok := msg.(tview.MouseMsg); ok && area.Contains(mouse.Position()) {
		pressed := mouse.Action == tview.MouseLeftDown
		if out != msg || pressed {
			return paneMsg{pane: p.pane, msg: out, focus: pressed}
		}
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

// paneView returns the view of target, marked so that clicking it focuses it.
func (m Model) paneView(target pane) tview.Widget {
	var child tview.Widget
	switch target {
	case paneGuildsTree:
		child = m.guildsTree.View()
	case paneMessagesList:
		child = m.messagesList.View()
	case paneComposer:
		child = m.composer.View()
	case paneMembersTree:
		child = m.membersTree.View()
	}
	return paneWidget{pane: target, child: child}
}

// paneFocused reports whether target has the focus and no picker is open.
func (m Model) paneFocused(target pane) bool {
	return m.focused == target && m.overlay == overlayNone
}

// setFocus focuses target if it can take the focus.
func (m *Model) setFocus(target pane) {
	if m.canFocus(target) {
		m.focused = target
	}
}

// canFocus reports whether target is shown and can take the focus.
func (m Model) canFocus(target pane) bool {
	switch target {
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

// typingFooter returns who is typing in the selected channel, or "" if nobody is.
func (m Model) typingFooter() string {
	channel := m.selectedChannel
	if channel == nil || len(m.typers) == 0 {
		return ""
	}

	var names []string
	for userID := range m.typers {
		if name := m.typerName(*channel, userID); name != "" {
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

// typerName returns the name of userID in channel, or "" if they are not known there.
func (m Model) typerName(channel discord.Channel, userID discord.UserID) string {
	if channel.GuildID.IsValid() {
		member, err := m.state.Cabinet.Member(channel.GuildID, userID)
		if err != nil {
			return ""
		}

		return cmp.Or(member.Nick, member.User.DisplayName, member.User.Username)
	}

	index := slices.IndexFunc(channel.DMRecipients, func(recipient discord.User) bool { return recipient.ID == userID })
	if index == -1 {
		return ""
	}

	return channel.DMRecipients[index].DisplayOrUsername()
}

var _ tview.Model[Model] = Model{}

func (m Model) Init() tview.Cmd {
	return tview.Batch(openState(m.state), listen(m.events), m.guildsTree.Init(), m.messagesList.Init(), tview.RequestTerminalInfo())
}

// Update returns m changed in response to msg and a command to run, or nil.
func (m Model) Update(msg tview.Msg) (Model, tview.Cmd) {
	m, cmd := m.update(msg)
	m.syncPanes()
	return m, cmd
}

// View shows the panes, with the autocomplete over the messages and an open picker on top.
func (m Model) View() tview.Widget {
	picker := m.pickerView()

	messages := m.paneView(paneMessagesList)
	if popup := m.composer.AutocompleteView(); popup != nil {
		messages = stack.New(messages, popup)
	}

	var right tview.Widget = column.New(messages, m.paneView(paneComposer))

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
