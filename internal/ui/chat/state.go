package chat

import (
	"log/slog"
	"slices"

	"github.com/ayn2op/discordo/internal/notifications"
	"github.com/ayn2op/tview"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/diamondburned/arikawa/v3/utils/httputil/httpdriver"
	"github.com/diamondburned/arikawa/v3/utils/ws"
	"github.com/diamondburned/ningen/v3"
	"github.com/diamondburned/ningen/v3/states/read"
)

func (m *Model) onRequest(r httpdriver.Request) error {
	if req, ok := r.(*httpdriver.DefaultRequest); ok {
		slog.Debug("new HTTP request", "method", req.Method, "url", req.URL)
	}
	return nil
}

func (m *Model) onRaw(event *ws.RawEvent) {
	slog.Debug(
		"new raw event",
		"code", event.OriginalCode,
		"type", event.OriginalType,
		// "data", event.Raw,
	)
}

func (m *Model) onReady(event *gateway.ReadyEvent) tview.Cmd {
	// Rebuild indexes from scratch so reconnects and account switches do not
	// retain pointers to detached tree nodes.
	m.guildsTree.resetNodeIndex()

	dmNode := tview.NewTreeNode("Direct Messages").SetReference(dmNode{}).SetExpandable(true).SetExpanded(false)
	m.guildsTree.dmRootNode = dmNode

	root := m.guildsTree.
		GetRoot().
		ClearChildren().
		AddChild(dmNode)

	// Track guilds already in folders to find orphans.
	// Newly joined guilds may not be synced to GuildFolders yet but always appear in guild positions.
	guildsInFolders := make(map[discord.GuildID]bool)
	for _, folder := range event.UserSettings.GuildFolders {
		for _, guildID := range folder.GuildIDs {
			guildsInFolders[guildID] = true
		}
	}

	guildsByID := make(map[discord.GuildID]*gateway.GuildCreateEvent, len(event.Guilds))
	for index := range event.Guilds {
		guildsByID[event.Guilds[index].ID] = &event.Guilds[index]
	}

	// Use GuildPositions for ordering (it's the canonical order).
	// Guilds not in any folder are "orphans" - add them directly to root.
	positions := event.UserSettings.GuildPositions
	// GuildPositions is normally set; fall back to event order if not.
	if len(positions) == 0 {
		positions = make([]discord.GuildID, 0, len(event.Guilds))
		for _, guildEvent := range event.Guilds {
			positions = append(positions, guildEvent.ID)
		}
	}

	for _, guildID := range positions {
		// Already handled in folder processing below.
		if guildsInFolders[guildID] {
			continue
		}

		if guildEvent, ok := guildsByID[guildID]; ok {
			m.guildsTree.createGuildNode(root, guildEvent.Guild)
		}
	}

	// Process folders (real folders and single-guild "folders")
	for _, folder := range event.UserSettings.GuildFolders {
		if folder.ID == 0 && len(folder.GuildIDs) == 1 {
			if guild, ok := guildsByID[folder.GuildIDs[0]]; ok {
				m.guildsTree.createGuildNode(root, guild.Guild)
			}
		} else {
			m.guildsTree.createFolderNode(folder, guildsByID)
		}
	}

	m.guildsTree.SetCurrentNode(root)
	return tview.SetFocus(m.guildsTree)
}

func (m *Model) onMessageCreate(message *gateway.MessageCreateEvent) tview.Cmd {
	selectedChannel, ok := m.SelectedChannel()
	if ok && selectedChannel.ID == message.ChannelID {
		m.removeTyper(message.Author.ID)
		m.messagesList.addMessage(message.Message)
		return nil
	}

	return m.notify(*message)
}

func (m *Model) notify(message gateway.MessageCreateEvent) tview.Cmd {
	return func() tview.Msg {
		if err := notifications.Notify(m.state, message, m.cfg); err != nil {
			slog.Error("failed to notify", "err", err, "channel_id", message.ChannelID, "message_id", message.ID)
		}
		return nil
	}
}

func (m *Model) onMessageUpdate(message *gateway.MessageUpdateEvent) {
	selectedChannel, ok := m.SelectedChannel()
	if !ok || selectedChannel.ID != message.ChannelID {
		return
	}

	index := slices.IndexFunc(m.messagesList.messages, func(m discord.Message) bool {
		return m.ID == message.ID
	})
	if index < 0 {
		return
	}

	m.messagesList.setMessage(index, message.Message)
}

func (m *Model) onMessageDelete(message *gateway.MessageDeleteEvent) {
	selectedChannel, ok := m.SelectedChannel()
	if !ok || selectedChannel.ID != message.ChannelID {
		return
	}

	prevCursor := m.messagesList.Cursor()
	deletedIndex := slices.IndexFunc(m.messagesList.messages, func(m discord.Message) bool {
		return m.ID == message.ID
	})
	if deletedIndex < 0 {
		return
	}

	m.messagesList.deleteMessage(deletedIndex)

	newCursor := cursorAfterDelete(prevCursor, deletedIndex, len(m.messagesList.messages))
	if newCursor != prevCursor {
		m.messagesList.SetCursor(newCursor)
	}
}

func cursorAfterDelete(prevCursor, deletedIndex, remaining int) int {
	switch {
	case prevCursor > deletedIndex:
		return prevCursor - 1
	case prevCursor == deletedIndex:
		if prev := deletedIndex - 1; prev >= 0 {
			return prev
		}
		return min(deletedIndex, remaining-1)
	default:
		return prevCursor
	}
}

func (m *Model) onGuildMembersChunk(event *gateway.GuildMembersChunkEvent) {
	m.messagesList.setFetchingChunk(false, uint(len(event.Members)))
}

func (m *Model) onGuildMemberRemove(event *gateway.GuildMemberRemoveEvent) {
	m.messageInput.cache.Invalidate(event.GuildID.String()+" "+event.User.Username, m.state.MemberState.SearchLimit)
}

func (m *Model) onTypingStart(event *gateway.TypingStartEvent) {
	selectedChannel, ok := m.SelectedChannel()
	if !ok || selectedChannel.ID != event.ChannelID {
		return
	}

	if m.isMe(event.UserID) {
		return
	}

	m.addTyper(event.UserID)
}

func (m *Model) onReadUpdate(event *read.UpdateEvent) {
	// Use indexed node lookup to avoid walking the whole tree on every read event.
	// This runs frequently while reading/typing across channels.
	if event.GuildID.IsValid() {
		if guildNode := m.guildsTree.findNodeByReference(event.GuildID); guildNode != nil {
			m.guildsTree.setNodeLineStyle(guildNode, m.guildsTree.guildNodeStyle(event.GuildID))
		}
	}

	// Channel style is always updated for the target channel regardless of
	// whether it's in a guild or DM.
	if channelNode := m.guildsTree.findNodeByReference(event.ChannelID); channelNode != nil {
		channel, err := m.state.Cabinet.Channel(event.ChannelID)
		if err != nil {
			indication := m.state.ChannelIsUnread(event.ChannelID, ningen.UnreadOpts{IncludeMutedCategories: true})
			m.guildsTree.setNodeLineStyle(channelNode, m.guildsTree.unreadStyle(indication))
			return
		}
		m.guildsTree.setNodeLineStyle(channelNode, m.guildsTree.channelNodeStyle(*channel))
	}
}
