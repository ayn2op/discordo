package messageslist

import (
	"context"
	"errors"
	"log/slog"
	"mime"
	"os"
	"path/filepath"
	"slices"

	"github.com/ayn2op/arikawa/v3/discord"
	"github.com/ayn2op/arikawa/v3/gateway"
	"github.com/ayn2op/discordo/internal/consts"
	"github.com/ayn2op/discordo/internal/ui"
	"github.com/ayn2op/tview"
	"github.com/ayn2op/tview/list"
	"github.com/ayn2op/tview/picker"
	"github.com/ayn2op/tview/viewport"
	"github.com/ncruces/zenity"
	"github.com/skratchdot/open-golang/open"
	"golang.design/x/clipboard"
)

// Msg is implemented by the messages the list sends itself, which must reach it whatever has the focus.
type Msg interface{ messagesList() }

type olderMessagesLoadedMsg struct {
	ChannelID discord.ChannelID
	Older     []discord.Message
}

func (olderMessagesLoadedMsg) messagesList() {}

// ReplyMsg asks to reply to Message by Name, mentioning its author if Mention is set.
type ReplyMsg struct {
	Message discord.Message
	Name    string
	Mention bool
}

// EditMsg asks to edit a message.
type EditMsg discord.Message

// ShowAttachmentsMsg asks to pick one of the attachments and links of the selected message.
type ShowAttachmentsMsg picker.Items

// listMsg moves the messages list.
type listMsg list.Change

// scrollMsg scrolls the messages list.
type scrollMsg viewport.Change

func (m *Model) selectUp() tview.Cmd {
	cursor := m.cursor()
	if cursor == -1 {
		m.selectBottom()
	} else if previous := m.messageIndex(cursor, -1); previous >= 0 {
		m.setCursor(previous)
		// Older messages are fetched once as the cursor reaches the top, rather than on each key press there.
		if m.atTop() {
			return m.fetchOlderMessages()
		}
	}
	return nil
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
		return olderMessagesLoadedMsg{ChannelID: channelID, Older: older}
	}
}

func (m Model) yankMessageID() tview.Cmd {
	selectedMessage, ok := m.selectedMessage()
	if !ok {
		return nil
	}

	return func() tview.Msg {
		if _, err := clipboard.Write(context.Background(), clipboard.FmtText, []byte(selectedMessage.ID.String())); err != nil {
			slog.Error("failed to write to clipboard", "err", err)
		}
		return nil
	}
}

func (m Model) yankContent() tview.Cmd {
	selectedMessage, ok := m.selectedMessage()
	if !ok {
		return nil
	}

	return func() tview.Msg {
		if _, err := clipboard.Write(context.Background(), clipboard.FmtText, []byte(selectedMessage.Content)); err != nil {
			slog.Error("failed to write to clipboard", "err", err)
		}
		return nil
	}
}

func (m Model) yankURL() tview.Cmd {
	selectedMessage, ok := m.selectedMessage()
	if !ok {
		return nil
	}

	return func() tview.Msg {
		if _, err := clipboard.Write(context.Background(), clipboard.FmtText, []byte(selectedMessage.URL())); err != nil {
			slog.Error("failed to write to clipboard", "err", err)
		}
		return nil
	}
}

func (m Model) open() tview.Cmd {
	return m.openWith(m.openAttachment)
}

func (m Model) openInBrowser() tview.Cmd {
	return m.openWith(func(attachment discord.Attachment) tview.Cmd { return openURL(attachment.URL) })
}

func (m Model) openWith(openAttachment func(discord.Attachment) tview.Cmd) tview.Cmd {
	selectedMessage, ok := m.selectedMessage()
	if !ok {
		return nil
	}

	urls := messageURLs(*selectedMessage)
	switch total := len(urls) + len(selectedMessage.Attachments); {
	case total == 0:
		return nil
	case total > 1:
		return m.showAttachmentsList(urls, selectedMessage.Attachments, openAttachment)
	case len(urls) == 1:
		return openURL(urls[0])
	}

	return openAttachment(selectedMessage.Attachments[0])
}

func (m Model) download() tview.Cmd {
	selectedMessage, ok := m.selectedMessage()
	if !ok || len(selectedMessage.Attachments) == 0 {
		return nil
	}
	if len(selectedMessage.Attachments) == 1 {
		attachment := selectedMessage.Attachments[0]
		return m.confirmAttachment(attachment, saveAttachment(attachment))
	}

	items := make(picker.Items, len(selectedMessage.Attachments))
	for i, attachment := range selectedMessage.Attachments {
		items[i] = picker.Item{Text: attachment.Filename, Reference: m.confirmAttachment(attachment, saveAttachment(attachment))}
	}
	return m.showAttachmentsPicker(items)
}

func (m Model) showAttachmentsList(urls []string, attachments []discord.Attachment, openAttachment func(discord.Attachment) tview.Cmd) tview.Cmd {
	var items picker.Items
	for _, attachment := range attachments {
		items = append(items, picker.Item{Text: attachment.Filename, Reference: openAttachment(attachment)})
	}
	for _, url := range urls {
		items = append(items, picker.Item{Text: url, Reference: openURL(url)})
	}
	return m.showAttachmentsPicker(items)
}

func (m Model) showAttachmentsPicker(items picker.Items) tview.Cmd {
	return func() tview.Msg { return ShowAttachmentsMsg(items) }
}

func (m Model) openAttachment(attachment discord.Attachment) tview.Cmd {
	return m.confirmAttachment(attachment, openDownloadedAttachment(attachment))
}

func (m Model) confirmAttachment(attachment discord.Attachment, action tview.Cmd) tview.Cmd {
	if !m.cfg.Attachments.AllowedMIMETypes.Has(attachment.ContentType) {
		return ui.ShowModal(
			"This attachment type is not allowed and may be unsafe. Continue anyway?",
			ui.ModalButton{Label: "No"},
			ui.ModalButton{Label: "Yes", Cmd: action},
		)
	}
	return action
}

func openDownloadedAttachment(attachment discord.Attachment) tview.Cmd {
	return func() tview.Msg {
		extension := filepath.Ext(attachment.Filename)
		if extension == "" {
			mediaType, _, _ := mime.ParseMediaType(attachment.ContentType)
			if extensions, _ := mime.ExtensionsByType(mediaType); len(extensions) != 0 {
				extension = extensions[0]
			}
		}

		dir := filepath.Join(consts.CacheDir(), "attachments")
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return attachmentErr("create attachments directory", err)
		}

		file, err := os.CreateTemp(dir, "attachment-*"+extension)
		if err != nil {
			return attachmentErr("create attachment file", err)
		}
		defer file.Close()

		path := file.Name()

		if err := downloadAttachment(attachment, path); err != nil {
			os.Remove(path)
			return attachmentErr("download attachment", err)
		} else if err := open.Start(path); err != nil {
			return attachmentErr("open attachment file", err)
		}

		return nil
	}
}

func saveAttachment(attachment discord.Attachment) tview.Cmd {
	return func() tview.Msg {
		destination, err := zenity.SelectFileSave(zenity.Filename(filepath.Base(attachment.Filename)), zenity.ConfirmOverwrite())
		if errors.Is(err, zenity.ErrCanceled) {
			return nil
		}
		if err != nil {
			return attachmentErr("select attachment destination", err)
		}

		if err := downloadAttachment(attachment, destination); err != nil {
			return attachmentErr("download attachment", err)
		}
		return nil
	}
}

func openURL(url string) tview.Cmd {
	return func() tview.Msg {
		if err := open.Start(url); err != nil {
			return attachmentErr("open URL", err)
		}
		return nil
	}
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
	return ui.ShowModal(
		"Are you sure you want to delete this message?",
		ui.ModalButton{Label: "Yes", Cmd: m.requestDelete(message)},
		ui.ModalButton{Label: "No"},
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
		// Do not fetch member for a webhook message.
		if message.WebhookID.IsValid() {
			continue
		}

		if member, _ := m.state.Cabinet.Member(guildID, message.Author.ID); member == nil {
			userID := message.Author.ID
			if _, ok := seen[userID]; !ok {
				seen[userID] = struct{}{}
				usersToFetch = append(usersToFetch, userID)
			}
		}
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
