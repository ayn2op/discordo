package chat

import (
	"fmt"
	"log/slog"

	"github.com/ayn2op/arikawa/v3/discord"
	"github.com/ayn2op/arikawa/v3/gateway"
	"github.com/ayn2op/arikawa/v3/utils/httputil/httpdriver"
	"github.com/ayn2op/arikawa/v3/utils/ws"
	"github.com/ayn2op/discordo/internal/ui"
	"github.com/ayn2op/ningen/v3/states/read"
	"github.com/ayn2op/tview"
)

func onRequest(req httpdriver.Request) error {
	if req, ok := req.(*httpdriver.DefaultRequest); ok {
		slog.Debug("new HTTP request", "method", req.Method, "url", req.URL)
	}

	return nil
}

func (m *Model) loadGuildsTree(event *gateway.ReadyEvent) tview.Cmd {
	cmd := m.updatePane(paneGuildsTree, event)
	m.setFocus(paneGuildsTree)
	return cmd
}

func (m *Model) addMessageOrNotify(message *gateway.MessageCreateEvent) tview.Cmd {
	cmd := m.updatePane(paneGuildsTree, message)

	if channel := m.selectedChannel; channel != nil && channel.ID == message.ChannelID {
		delete(m.typers, message.Author.ID)
		cmd = tview.Batch(cmd, m.messagesList.AddMessage(message.Message))
		if !m.windowUnfocused || !m.cfg.Notifications.WhenUnfocused {
			return cmd
		}
	}
	return tview.Batch(cmd, m.notify(*message))
}

func (m Model) notify(message gateway.MessageCreateEvent) tview.Cmd {
	return func() tview.Msg {
		return m.notification(message)
	}
}

// notification shows a desktop notification for message if it mentions the user.
func (m Model) notification(message gateway.MessageCreateEvent) tview.Msg {
	if !m.cfg.Notifications.Enabled || m.cfg.Status == discord.DoNotDisturbStatus {
		return nil
	}

	if m.state.MessageMentions(&message.Message) == 0 {
		return nil
	}

	content := message.Content
	if content == "" && len(message.Attachments) > 0 {
		content = "Uploaded " + message.Attachments[0].Filename
	}

	if content == "" {
		return nil
	}

	title, err := m.notificationTitle(message)
	if err != nil {
		slog.Error("failed to get notification title", "err", err)
		return nil
	}

	return tview.Notify(title, content)()
}

// notificationTitle returns the author of message, followed by its channel and guild for a guild message.
func (m Model) notificationTitle(message gateway.MessageCreateEvent) (string, error) {
	title := message.Author.DisplayOrUsername()
	channel, err := m.state.Cabinet.Channel(message.ChannelID)
	if err != nil {
		return "", fmt.Errorf("failed to get channel %s from state: %w", message.ChannelID, err)
	}

	if !channel.GuildID.IsValid() {
		return title, nil
	}

	guild, err := m.state.Cabinet.Guild(channel.GuildID)
	if err != nil {
		return "", fmt.Errorf("failed to get guild %s from state: %w", channel.GuildID, err)
	}

	if member := message.Member; member != nil && member.Nick != "" {
		title = member.Nick
	}

	return title + " (#" + channel.Name + ", " + guild.Name + ")", nil
}

func (m Model) refreshMemberNames(event *gateway.GuildMembersChunkEvent) tview.Cmd {
	m.messagesList.InvalidateRendered()
	return m.composer.CacheMemberSearch(event)
}

func (m Model) showTypingIndicator(event *gateway.TypingStartEvent) tview.Cmd {
	if channel := m.selectedChannel; channel == nil || channel.ID != event.ChannelID || ui.IsMe(m.state, event.UserID) {
		return nil
	}
	return m.addTyper(event.UserID)
}

// applyEvent updates the model with a gateway event and returns what the event needs done.
func (m *Model) applyEvent(event gateway.Event) tview.Cmd {
	switch event := event.(type) {
	case *ws.RawEvent:
		slog.Debug("new raw event", "code", event.OriginalCode, "type", event.OriginalType)

	case *gateway.ReadyEvent:
		return tview.Batch(m.loadGuildsTree(event), m.updatePane(paneMembersTree, event))

	case *gateway.MessageCreateEvent:
		return m.addMessageOrNotify(event)
	case *gateway.MessageUpdateEvent:
		return m.messagesList.UpdateMessage(event.Message)
	case *gateway.PresenceUpdateEvent:
		return tview.Batch(m.updatePane(paneGuildsTree, event), m.updatePane(paneMembersTree, event))
	case *gateway.GuildMemberListUpdateEvent:
		return m.updatePane(paneMembersTree, event)
	case *gateway.MessageDeleteEvent:
		m.messagesList.DeleteMessage(event.ChannelID, event.ID)
	case *gateway.MessageReactionAddEvent:
		return m.messagesList.RefreshMessage(event.ChannelID, event.MessageID)
	case *gateway.MessageReactionAddManyEvent:
		return m.messagesList.RefreshMessage(event.ChannelID, event.MessageID)
	case *gateway.MessageReactionRemoveEvent:
		return m.messagesList.RefreshMessage(event.ChannelID, event.MessageID)
	case *gateway.MessageReactionRemoveAllEvent:
		return m.messagesList.RefreshMessage(event.ChannelID, event.MessageID)
	case *gateway.MessageReactionRemoveEmojiEvent:
		return m.messagesList.RefreshMessage(event.ChannelID, event.MessageID)

	case *gateway.GuildMembersChunkEvent:
		return m.refreshMemberNames(event)
	case *gateway.GuildMemberRemoveEvent:
		m.composer.OnGuildMemberRemove(event)

	case *gateway.TypingStartEvent:
		if m.cfg.TypingIndicator.Receive {
			return m.showTypingIndicator(event)
		}

	case *read.UpdateEvent:
		return m.updatePane(paneGuildsTree, event)
	case *gateway.VoiceStateUpdateEvent:
		return m.updatePane(paneGuildsTree, event)
	}
	return nil
}
