package chat

import (
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/ayn2op/arikawa/v3/discord"
	"github.com/ayn2op/arikawa/v3/gateway"
	"github.com/ayn2op/arikawa/v3/session"
	"github.com/ayn2op/arikawa/v3/state"
	"github.com/ayn2op/arikawa/v3/state/store/defaultstore"
	"github.com/ayn2op/arikawa/v3/utils/handler"
	"github.com/ayn2op/arikawa/v3/utils/httputil"
	"github.com/ayn2op/arikawa/v3/utils/ws"
	"github.com/ayn2op/discordo/internal/config"
	clientgateway "github.com/ayn2op/discordo/internal/gateway"
	"github.com/ayn2op/discordo/internal/http"
	"github.com/ayn2op/discordo/internal/ui"
	"github.com/ayn2op/discordo/internal/ui/chat/attachmentspicker"
	"github.com/ayn2op/discordo/internal/ui/chat/channelspicker"
	"github.com/ayn2op/ningen/v3"
	"github.com/ayn2op/ningen/v3/states/read"
	"github.com/ayn2op/tview"
	"github.com/ayn2op/tview/flex"
	"github.com/ayn2op/tview/keybind"
	"github.com/ayn2op/tview/layers"
	"github.com/gdamore/tcell/v3"
)

const typingDuration = 10 * time.Second

const (
	flexLayerName         = "flex"
	mentionsListLayerName = "mentionsList"

	channelsPickerLayerName    = "channelsPicker"
	attachmentsPickerLayerName = "attachmentsPicker"
)

type Model struct {
	*layers.Layers

	// guildsTree (sidebar) + rightFlex
	mainFlex *flex.Model
	// messagesList + composer
	rightFlex *flex.Model

	guildsTree     *guildsTree
	messagesList   *messagesList
	composer       *composer
	channelsPicker *channelspicker.Model
	focused        tview.Model

	selectedChannel   *discord.Channel
	selectedChannelMu sync.RWMutex

	state  *ningen.State
	events chan gateway.Event

	typersMu sync.RWMutex
	typers   map[discord.UserID]*time.Timer

	cfg *config.Config
}

func NewModel(cfg *config.Config, token string) *Model {
	m := &Model{
		Layers: layers.New(),

		mainFlex:  flex.NewModel(),
		rightFlex: flex.NewModel(),

		typers: make(map[discord.UserID]*time.Timer),

		cfg: cfg,
	}

	m.guildsTree = newGuildsTree(cfg, m)
	m.messagesList = newMessagesList(cfg, m)
	m.composer = newComposer(cfg, m)
	m.channelsPicker = channelspicker.NewModel(cfg)

	id := gateway.NewIdentifier(gateway.IdentifyCommand{
		Token:      token,
		Properties: http.IdentifyProperties(),
	})

	session := session.NewWithGateway(clientgateway.New(id), handler.New())
	session.Client = http.NewClient(token)
	state := state.NewFromSession(session, defaultstore.New())
	m.state = ningen.FromState(state)
	m.state.DontWaitForReady = true

	m.events = make(chan gateway.Event)
	m.state.AddHandler(m.events)
	m.state.StateLog = func(err error) {
		slog.Error("state log", "err", err)
	}
	m.state.OnRequest = append(m.state.OnRequest, httputil.WithHeaders(http.Headers()), m.onRequest)

	m.SetBackgroundLayerStyle(m.cfg.Theme.Dialog.BackgroundStyle.Style)
	m.buildLayout()
	return m
}

func (m *Model) SelectedChannel() (*discord.Channel, bool) {
	m.selectedChannelMu.RLock()
	defer m.selectedChannelMu.RUnlock()
	return m.selectedChannel, m.selectedChannel != nil
}

func (m *Model) SetSelectedChannel(channel *discord.Channel) {
	m.selectedChannelMu.Lock()
	m.selectedChannel = channel
	m.selectedChannelMu.Unlock()
}

func (m *Model) isMe(id discord.UserID) bool {
	me, _ := m.state.Cabinet.Me()
	return me != nil && id == me.ID
}

func (m *Model) buildLayout() {
	m.Clear()
	m.rightFlex.Clear()
	m.mainFlex.Clear()

	m.rightFlex.
		SetDirection(flex.DirectionRow).
		AddItem(m.messagesList, 0, 1, false).
		AddItem(m.composer, 3, 1, false)
	// The guilds tree is always focused first at start-up.
	m.mainFlex.
		AddItem(m.guildsTree, 0, m.cfg.Sidebar.WidthPercent, true).
		AddItem(m.rightFlex, 0, 100-m.cfg.Sidebar.WidthPercent, false)

	m.AddLayer(m.mainFlex, layers.WithName(flexLayerName), layers.WithResize(true), layers.WithVisible(true))
	m.AddLayer(
		m.composer.mentionsList,
		layers.WithName(mentionsListLayerName),
		layers.WithResize(false),
		layers.WithVisible(false),
		layers.WithEnabled(false),
	)
}

func (m *Model) togglePicker() tview.Cmd {
	if m.HasLayer(channelsPickerLayerName) {
		return m.closePicker()
	}
	return m.openPicker()
}

func (m *Model) openPicker() tview.Cmd {
	m.AddLayer(
		ui.Centered(m.channelsPicker, m.cfg.Picker.Width, m.cfg.Picker.Height),
		layers.WithName(channelsPickerLayerName),
		layers.WithResize(true),
		layers.WithVisible(true),
		layers.WithOverlay(),
	).SendToFront(channelsPickerLayerName)
	m.channelsPicker.RefreshChannels(m.state)
	return tview.SetFocus(m.channelsPicker)
}

func (m *Model) closePicker() tview.Cmd {
	m.RemoveLayer(channelsPickerLayerName)
	m.channelsPicker.Refresh()
	return tview.SetFocus(m.mainFlex)
}

func (m *Model) closeAttachmentsPicker() tview.Cmd {
	m.RemoveLayer(attachmentsPickerLayerName)
	return tview.SetFocus(m.messagesList)
}

func (m *Model) navigateToChannel(channelID discord.ChannelID) tview.Cmd {
	channel, err := m.state.Cabinet.Channel(channelID)
	if err != nil {
		slog.Error("failed to get channel from state", "err", err, "channel_id", channelID)
		return nil
	}

	node := m.guildsTree.findNodeByChannelID(channel.ID)
	if node == nil {
		slog.Error("failed to locate channel in tree", "channel_id", channel.ID)
		return nil
	}

	m.guildsTree.expandPathToNode(node)
	m.guildsTree.SetCurrentNode(node)
	focus := m.closePicker()
	if channel.Type != discord.GuildCategory {
		return tview.Sequence(focus, m.guildsTree.onSelected(node))
	}
	return focus
}

func (m *Model) toggleGuildsTree() tview.Cmd {
	if m.mainFlex.GetItemCount() == 2 {
		m.mainFlex.RemoveItem(m.guildsTree)
		if m.guildsTree.HasFocus() {
			return tview.SetFocus(m.mainFlex)
		}
	} else {
		m.buildLayout()
		return tview.SetFocus(m.guildsTree)
	}
	return nil
}

func (m *Model) focusGuildsTree() tview.Cmd {
	if m.mainFlex.GetItemCount() == 2 {
		return tview.SetFocus(m.guildsTree)
	}
	return nil
}

func (m *Model) focusComposer() tview.Cmd {
	if !m.composer.Disabled() {
		return tview.SetFocus(m.composer)
	}
	return nil
}

func (m *Model) focusPrevious() tview.Cmd {
	switch m.focused {
	case m.guildsTree:
		if cmd := m.focusComposer(); cmd != nil {
			return cmd
		}
		return tview.SetFocus(m.messagesList)
	case m.messagesList:
		if cmd := m.focusGuildsTree(); cmd != nil {
			return cmd
		}
		if cmd := m.focusComposer(); cmd != nil {
			return cmd
		}
		return tview.SetFocus(m.messagesList)
	case m.composer:
		return tview.SetFocus(m.messagesList)
	}
	return nil
}

func (m *Model) focusNext() tview.Cmd {
	switch m.focused {
	case m.guildsTree:
		return tview.SetFocus(m.messagesList)
	case m.messagesList:
		if cmd := m.focusComposer(); cmd != nil {
			return cmd
		}
		if cmd := m.focusGuildsTree(); cmd != nil {
			return cmd
		}
	case m.composer:
		if cmd := m.focusGuildsTree(); cmd != nil {
			return cmd
		}
		return tview.SetFocus(m.messagesList)
	}
	return nil
}

func (m *Model) Update(msg tview.Msg) tview.Cmd {
	switch msg := msg.(type) {
	case FocusedMsg:
		m.focused = msg.Model
		return nil
	case tview.InitMsg:
		return tview.Batch(openState(m.state), listen(m.events))
	case gateway.Event:
		switch eventMsg := msg.(type) {
		case *ws.RawEvent:
			m.onRaw(eventMsg)

		case *gateway.ReadyEvent:
			return tview.Batch(m.onReady(eventMsg), listen(m.events))

		case *gateway.MessageCreateEvent:
			return tview.Batch(m.onMessageCreate(eventMsg), listen(m.events))
		case *gateway.MessageUpdateEvent:
			m.onMessageUpdate(eventMsg)
		case *gateway.PresenceUpdateEvent:
			m.onPresenceUpdate(eventMsg)
		case *gateway.MessageDeleteEvent:
			m.onMessageDelete(eventMsg)
		case *gateway.MessageReactionAddEvent:
			m.onMessageReaction(eventMsg.ChannelID, eventMsg.MessageID)
		case *gateway.MessageReactionRemoveEvent:
			m.onMessageReaction(eventMsg.ChannelID, eventMsg.MessageID)
		case *gateway.MessageReactionRemoveAllEvent:
			m.onMessageReaction(eventMsg.ChannelID, eventMsg.MessageID)
		case *gateway.MessageReactionRemoveEmojiEvent:
			m.onMessageReaction(eventMsg.ChannelID, eventMsg.MessageID)

		case *gateway.GuildMembersChunkEvent:
			return tview.Batch(m.onGuildMembersChunk(eventMsg), listen(m.events))
		case *gateway.GuildMemberRemoveEvent:
			m.onGuildMemberRemove(eventMsg)

		case *gateway.TypingStartEvent:
			if m.cfg.TypingIndicator.Receive {
				m.onTypingStart(eventMsg)
			}

		case *read.UpdateEvent:
			m.onReadUpdate(eventMsg)
		}
		return listen(m.events)
	case channelLoadedMsg:
		node := m.guildsTree.CurrentNode()
		if node == nil {
			return nil
		}
		channelID, ok := node.Reference().(discord.ChannelID)
		if !ok || channelID != msg.Channel.ID {
			return nil
		}

		m.SetSelectedChannel(&msg.Channel)
		m.clearTypers()
		m.composer.typingUntil = time.Time{}

		m.messagesList.reset()
		m.messagesList.setTitle(msg.Channel)
		m.messagesList.setMessages(msg.Messages)
		m.messagesList.ScrollBottom()

		isDM := msg.Channel.Type == discord.DirectMessage || msg.Channel.Type == discord.GroupDM
		hasNoPerm := !isDM && !m.state.HasPermissions(msg.Channel.ID, discord.PermissionSendMessages)
		m.composer.SetDisabled(hasNoPerm)

		text := "Message..."
		var focusCmd tview.Cmd

		if hasNoPerm {
			text = "You do not have permission to send messages in this channel."
		} else if m.cfg.AutoFocus {
			focusCmd = m.focusComposer()
		}
		m.composer.SetPlaceholder(tview.NewLine(tview.NewSegment(text, tcell.StyleDefault.Dim(true))))
		if msg.Channel.GuildID.IsValid() {
			return tview.Batch(focusCmd, m.messagesList.requestGuildMembers(msg.Channel.GuildID, msg.Messages))
		}
		return focusCmd
	case deleteMessageMsg:
		return m.messagesList.deleteMessageRequest(discord.Message(msg))
	case channelspicker.SelectedMsg:
		return m.navigateToChannel(msg.ChannelID)
	case channelspicker.CancelMsg:
		return m.closePicker()
	case attachmentspicker.SelectedMsg:
		return tview.Sequence(msg.Open, m.closeAttachmentsPicker())
	case attachmentspicker.CancelMsg:
		return m.closeAttachmentsPicker()
	case QuitMsg:
		return closeState(m.state)
	case tview.KeyMsg:
		switch {
		case keybind.Matches(msg, m.cfg.Keybinds.FocusGuildsTree.Keybind):
			m.composer.removeMentionsList()
			return m.focusGuildsTree()
		case keybind.Matches(msg, m.cfg.Keybinds.FocusMessagesList.Keybind):
			m.composer.removeMentionsList()
			return tview.SetFocus(m.messagesList)
		case keybind.Matches(msg, m.cfg.Keybinds.FocusComposer.Keybind):
			return m.focusComposer()

		case keybind.Matches(msg, m.cfg.Keybinds.FocusPrevious.Keybind):
			return m.focusPrevious()
		case keybind.Matches(msg, m.cfg.Keybinds.FocusNext.Keybind):
			return m.focusNext()

		case keybind.Matches(msg, m.cfg.Keybinds.ToggleGuildsTree.Keybind):
			return m.toggleGuildsTree()
		case keybind.Matches(msg, m.cfg.Keybinds.ToggleChannelsPicker.Keybind):
			return m.togglePicker()

		case keybind.Matches(msg, m.cfg.Keybinds.Logout.Keybind):
			return tview.Sequence(closeState(m.state), logout())
		}
	case tabSuggestMsg:
		return m.composer.Update(msg)
	}
	return m.Layers.Update(msg)
}

func (m *Model) clearTypers() {
	m.typersMu.Lock()
	for _, timer := range m.typers {
		timer.Stop()
	}
	clear(m.typers)
	m.typersMu.Unlock()
	m.updateFooter()
}

func (m *Model) addTyper(userID discord.UserID) {
	m.typersMu.Lock()
	typer, ok := m.typers[userID]
	if ok {
		typer.Reset(typingDuration)
	} else {
		m.typers[userID] = time.AfterFunc(typingDuration, func() {
			m.removeTyper(userID)
		})
	}
	m.typersMu.Unlock()
	m.updateFooter()
}

func (m *Model) removeTyper(userID discord.UserID) {
	m.typersMu.Lock()
	if typer, ok := m.typers[userID]; ok {
		typer.Stop()
		delete(m.typers, userID)
	}
	m.typersMu.Unlock()
	m.updateFooter()
}

func (m *Model) updateFooter() {
	selectedChannel, ok := m.SelectedChannel()
	if !ok {
		return
	}
	guildID := selectedChannel.GuildID

	m.typersMu.RLock()
	defer m.typersMu.RUnlock()

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
