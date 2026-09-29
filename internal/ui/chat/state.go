package chat

import (
	"log/slog"

	"github.com/ayn2op/arikawa/v3/discord"
	"github.com/ayn2op/arikawa/v3/gateway"
	"github.com/ayn2op/arikawa/v3/utils/httputil/httpdriver"
	"github.com/ayn2op/arikawa/v3/utils/ws"
	"github.com/ayn2op/discordo/internal/ui"
	"github.com/ayn2op/ningen/v3/states/read"
	"github.com/ayn2op/tview"
)

func onRequest(r httpdriver.Request) error {
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

func (m *Model) loadGuildsTree(event *gateway.ReadyEvent) tview.Cmd {
	cmd := m.updatePane(guildsTreePane, event)
	m.setFocus(guildsTreePane)
	return cmd
}

func (m *Model) addMessageOrNotify(message *gateway.MessageCreateEvent) tview.Cmd {
	cmd := m.updatePane(guildsTreePane, message)

	if channel := m.selectedChannel; channel != nil && channel.ID == message.ChannelID {
		m.removeTyper(message.Author.ID)
		cmd = tview.Batch(cmd, m.messagesList.AddMessage(message.Message))
		if !m.windowUnfocused || !m.cfg.Notifications.WhenUnfocused {
			return cmd
		}
	}
	return tview.Batch(cmd, m.notify(*message))
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

func (m *Model) refreshMemberNames(event *gateway.GuildMembersChunkEvent) tview.Cmd {
	m.messagesList.InvalidateRendered()
	return m.composer.CacheMemberSearch(event)
}

func (m *Model) showTypingIndicator(event *gateway.TypingStartEvent) tview.Cmd {
	if channel := m.selectedChannel; channel == nil || channel.ID != event.ChannelID || ui.IsMe(m.state, event.UserID) {
		return nil
	}
	return m.addTyper(event.UserID)
}

// applyEvent updates the model with a gateway event and returns what the event needs done.
func (m *Model) applyEvent(event gateway.Event) tview.Cmd {
	switch event := event.(type) {
	case *ws.RawEvent:
		m.onRaw(event)

	case *gateway.ReadyEvent:
		return tview.Batch(m.loadGuildsTree(event), m.updatePane(membersTreePane, event))

	case *gateway.MessageCreateEvent:
		return m.addMessageOrNotify(event)
	case *gateway.MessageUpdateEvent:
		return m.messagesList.UpdateMessage(event.Message)
	case *gateway.PresenceUpdateEvent:
		return tview.Batch(m.updatePane(guildsTreePane, event), m.updatePane(membersTreePane, event))
	case *gateway.GuildMemberListUpdateEvent:
		return m.updatePane(membersTreePane, event)
	case *gateway.MessageDeleteEvent:
		m.messagesList.DeleteMessage(event.ChannelID, event.ID)
	case *gateway.MessageReactionAddEvent:
		m.messagesList.RefreshMessage(event.ChannelID, event.MessageID)
	case *gateway.MessageReactionAddManyEvent:
		m.messagesList.RefreshMessage(event.ChannelID, event.MessageID)
	case *gateway.MessageReactionRemoveEvent:
		m.messagesList.RefreshMessage(event.ChannelID, event.MessageID)
	case *gateway.MessageReactionRemoveAllEvent:
		m.messagesList.RefreshMessage(event.ChannelID, event.MessageID)
	case *gateway.MessageReactionRemoveEmojiEvent:
		m.messagesList.RefreshMessage(event.ChannelID, event.MessageID)

	case *gateway.GuildMembersChunkEvent:
		return m.refreshMemberNames(event)
	case *gateway.GuildMemberRemoveEvent:
		m.composer.OnGuildMemberRemove(event)

	case *gateway.TypingStartEvent:
		if m.cfg.TypingIndicator.Receive {
			return m.showTypingIndicator(event)
		}

	case *read.UpdateEvent:
		return m.updatePane(guildsTreePane, event)
	}
	return nil
}
