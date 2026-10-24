package messageslist

import (
	"context"
	"log/slog"
	"slices"

	"github.com/ayn2op/arikawa/v3/discord"
	"github.com/ayn2op/arikawa/v3/gateway"
	"github.com/ayn2op/discordo/internal/ui"
	"github.com/ayn2op/tview"
	"github.com/ayn2op/tview/viewport"
)

// Msg is implemented by the messages the list sends itself, which must reach it whatever has the focus.
type Msg interface{ messagesList() }

type olderMessagesLoadedMsg struct {
	channelID discord.ChannelID
	older     []discord.Message
}

func (olderMessagesLoadedMsg) messagesList() {}

// voiceMsg reports that the status of the voice message player changed.
type voiceMsg struct{}

func (voiceMsg) messagesList() {}

// Init waits for that.
func (m Model) Init() tview.Cmd {
	return func() tview.Msg {
		<-m.voicePlayer.Updates()
		return voiceMsg{}
	}
}

// ReplyMsg asks to reply to Message by Name, mentioning its author if Mention is set.
type ReplyMsg struct {
	Message discord.Message
	Name    string
	Mention bool
}

// EditMsg asks to edit a message.
type EditMsg discord.Message

// listMsg is the item of the messages list to move the cursor to.
type listMsg int

// scrollMsg scrolls the messages list.
type scrollMsg viewport.ScrollState

func (m *Model) selectUp() tview.Cmd {
	if m.cursor == -1 {
		m.selectBottom()
		return nil
	}

	previous := m.messageIndex(m.cursor, -1)
	if previous < 0 {
		return nil
	}

	m.setCursor(previous)
	// Older messages are fetched once as the cursor reaches the top, rather than on each key press there.
	if !m.atTop() {
		return nil
	}

	return m.fetchOlderMessages()
}

func (m Model) fetchOlderMessages() tview.Cmd {
	selectedChannel := m.channel
	if selectedChannel == nil {
		return nil
	}

	channelID := selectedChannel.ID
	first := m.messageIndex(-1, 1)
	if first == -1 {
		return nil
	}
	before := m.items[first].message.ID
	limit := uint(m.cfg.MessagesLimit)
	return func() tview.Msg {
		messages, err := m.state.MessagesBefore(channelID, before, limit)
		if err != nil {
			slog.Error("failed to fetch older messages", "err", err)
			return nil
		}
		if len(messages) == 0 {
			return nil
		}

		older := slices.Clone(messages)
		slices.Reverse(older)
		return olderMessagesLoadedMsg{channelID: channelID, older: older}
	}
}

func (m Model) yankMessageID() tview.Cmd {
	selectedMessage, ok := m.selectedMessage()
	if !ok {
		return nil
	}

	return ui.WriteClipboard(selectedMessage.ID.String())
}

func (m Model) yankContent() tview.Cmd {
	selectedMessage, ok := m.selectedMessage()
	if !ok {
		return nil
	}

	return ui.WriteClipboard(selectedMessage.Content)
}

func (m Model) yankURL() tview.Cmd {
	selectedMessage, ok := m.selectedMessage()
	if !ok {
		return nil
	}

	return ui.WriteClipboard(selectedMessage.URL())
}

func (m Model) reply(mention bool) tview.Cmd {
	selectedMessage, ok := m.selectedMessage()
	if !ok {
		return nil
	}

	name := selectedMessage.Author.DisplayOrUsername()
	if member := m.memberForMessage(*selectedMessage); member != nil && member.Nick != "" {
		name = member.Nick
	}

	message := *selectedMessage
	return func() tview.Msg { return ReplyMsg{Message: message, Name: name, Mention: mention} }
}

func (m Model) editSelectedMessage() tview.Cmd {
	selectedMessage, ok := m.selectedMessage()
	if !ok {
		return nil
	}

	if !ui.IsMe(m.state, selectedMessage.Author.ID) {
		slog.Error("failed to edit message; not the author", "channel_id", selectedMessage.ChannelID, "message_id", selectedMessage.ID)
		return nil
	}

	message := *selectedMessage
	return func() tview.Msg { return EditMsg(message) }
}

func (m Model) confirmDelete() tview.Cmd {
	selectedMessage, ok := m.selectedMessage()
	if !ok || !m.canDeleteMessage(*selectedMessage) {
		return nil
	}
	message := *selectedMessage
	return ui.ShowDialog(
		"Are you sure you want to delete this message?",
		ui.DialogButton{Label: "No"},
		ui.DialogButton{Label: "Yes", Cmd: m.requestDelete(message)},
	)
}

func (m Model) deleteSelectedMessage() tview.Cmd {
	selectedMessage, ok := m.selectedMessage()
	if !ok {
		return nil
	}
	return m.requestDelete(*selectedMessage)
}

func (m Model) requestDelete(message discord.Message) tview.Cmd {
	return func() tview.Msg {
		if !m.canDeleteMessage(message) {
			slog.Error("failed to delete message; missing relevant permissions", "channel_id", message.ChannelID, "message_id", message.ID)
			return nil
		}

		if err := m.state.DeleteMessage(message.ChannelID, message.ID, ""); err != nil {
			slog.Error("failed to delete message", "channel_id", message.ChannelID, "message_id", message.ID, "err", err)
			return nil
		}

		if err := m.state.MessageRemove(message.ChannelID, message.ID); err != nil {
			slog.Error("failed to delete message", "channel_id", message.ChannelID, "message_id", message.ID, "err", err)
		}
		return nil
	}
}

func (m Model) requestGuildMembers(guildID discord.GuildID, messages []discord.Message) tview.Cmd {
	usersToFetch := make([]discord.UserID, 0, len(messages))
	seen := make(map[discord.UserID]struct{}, len(messages))

	for _, message := range messages {
		userID := message.Author.ID
		// Do not fetch member for a webhook message.
		if message.WebhookID.IsValid() {
			continue
		}

		if _, ok := seen[userID]; ok {
			continue
		}

		if member, _ := m.state.Cabinet.Member(guildID, userID); member != nil {
			continue
		}

		seen[userID] = struct{}{}
		usersToFetch = append(usersToFetch, userID)
	}

	if len(usersToFetch) == 0 {
		return nil
	}

	return func() tview.Msg {
		if err := m.state.SendGateway(context.Background(), &gateway.RequestGuildMembersCommand{
			GuildIDs: []discord.GuildID{guildID},
			UserIDs:  usersToFetch,
		}); err != nil {
			slog.Error("failed to request guild members", "guild_id", guildID, "err", err)
		}
		return nil
	}
}
