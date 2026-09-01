package chat

import (
	"log/slog"
	"slices"

	"github.com/ayn2op/arikawa/v3/discord"
	"github.com/ayn2op/arikawa/v3/gateway"
	"github.com/ayn2op/arikawa/v3/utils/httputil/httpdriver"
	"github.com/ayn2op/arikawa/v3/utils/ws"
	"github.com/ayn2op/ningen/v3/states/read"
	"github.com/ayn2op/tview"
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
	m.guildsTree.Update(event)
	m.setFocus(m.guildsTree)
	return nil
}

func (m *Model) onMessageCreate(message *gateway.MessageCreateEvent) tview.Cmd {
	m.guildsTree.Update(message)

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
		if !m.cfg.Notifications.Enabled || m.cfg.Status == discord.DoNotDisturbStatus {
			return nil
		}

		mentions := m.state.MessageMentions(&message.Message)
		if mentions == 0 {
			return nil
		}

		// Handle sent files
		content := message.Content
		if message.Content == "" && len(message.Attachments) > 0 {
			content = "Uploaded " + message.Attachments[0].Filename
		}

		if content == "" {
			return nil
		}

		title := message.Author.DisplayOrUsername()
		channel, err := m.state.Cabinet.Channel(message.ChannelID)
		if err != nil {
			slog.Error("failed to get channel from state", "err", err, "channel_id", message.ChannelID)
			return nil
		}

		if channel.GuildID.IsValid() {
			guild, err := m.state.Cabinet.Guild(channel.GuildID)
			if err != nil {
				slog.Error("failed to get guild from state", "err", err, "guild_id", channel.GuildID)
				return nil
			}

			if member := message.Member; member != nil && member.Nick != "" {
				title = member.Nick
			}

			title += " (#" + channel.Name + ", " + guild.Name + ")"
		}

		return tview.Notify(title, content)()
	}
}

func (m *Model) onPresenceUpdate(presence *gateway.PresenceUpdateEvent) {
	m.guildsTree.Update(presence)
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

func (m *Model) onMessageReaction(channelID discord.ChannelID, messageID discord.MessageID) {
	selectedChannel, ok := m.SelectedChannel()
	if !ok || selectedChannel.ID != channelID {
		return
	}

	index := slices.IndexFunc(m.messagesList.messages, func(message discord.Message) bool {
		return message.ID == messageID
	})
	message, err := m.state.Cabinet.Message(channelID, messageID)
	if index >= 0 && err == nil {
		m.messagesList.setMessage(index, *message)
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

func (m *Model) onGuildMembersChunk(event *gateway.GuildMembersChunkEvent) tview.Cmd {
	m.messagesList.invalidateRenderedMessages()
	return m.composer.onGuildMembersChunk(event)
}

func (m *Model) onGuildMemberRemove(event *gateway.GuildMemberRemoveEvent) {
	m.composer.cache.Invalidate(event.GuildID.String()+" "+event.User.Username, m.state.MemberState.SearchLimit)
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
	m.guildsTree.Update(event)
}
