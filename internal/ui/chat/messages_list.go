package chat

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/ayn2op/discordo/internal/clipboard"
	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/discordo/internal/consts"
	"github.com/ayn2op/discordo/internal/markdown"
	"github.com/ayn2op/discordo/internal/ui"
	"github.com/ayn2op/tview"
	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/diamondburned/arikawa/v3/state"
	"github.com/diamondburned/arikawa/v3/utils/json/option"
	"github.com/diamondburned/ningen/v3/discordmd"
	"github.com/gdamore/tcell/v3"
	"github.com/skratchdot/open-golang/open"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

type messagesList struct {
	*tview.ScrollList
	cfg      *config.Config
	chatView *View
	messages []discord.Message

	renderer *markdown.Renderer

	fetchingMembers struct {
		mu    sync.Mutex
		value bool
		count uint
		done  chan struct{}
	}
}

func newMessagesList(cfg *config.Config, chatView *View) *messagesList {
	ml := &messagesList{
		ScrollList: tview.NewScrollList(),
		cfg:        cfg,
		chatView:   chatView,
		renderer:   markdown.NewRenderer(cfg.Theme.MessagesList),
	}

	ml.Box = ui.ConfigureBox(ml.Box, &cfg.Theme)
	ml.SetTitle("Messages")
	ml.SetBuilder(ml.buildItem)
	ml.SetTrackEnd(true)
	ml.SetInputCapture(ml.onInputCapture)
	return ml
}

func (ml *messagesList) reset() {
	ml.messages = nil
	ml.
		Clear().
		SetBuilder(ml.buildItem).
		SetTitle("")
}

func (ml *messagesList) setTitle(channel discord.Channel) {
	title := ui.ChannelToString(channel)
	if topic := channel.Topic; topic != "" {
		title += " - " + topic
	}

	ml.SetTitle(title)
}

func (ml *messagesList) drawMessages(messages []discord.Message) {
	ml.messages = slices.Grow(ml.messages[:0], len(messages))
	ml.messages = ml.messages[:len(messages)]
	copy(ml.messages, messages)
	slices.Reverse(ml.messages)
}

func (ml *messagesList) addMessage(message discord.Message) {
	ml.messages = append(ml.messages, message)
}

func (ml *messagesList) setMessage(index int, message discord.Message) {
	if index < 0 || index >= len(ml.messages) {
		return
	}

	ml.messages[index] = message
}

func (ml *messagesList) deleteMessage(index int) {
	if index < 0 || index >= len(ml.messages) {
		return
	}

	ml.messages = append(ml.messages[:index], ml.messages[index+1:]...)
}

func (ml *messagesList) clearSelection() {
	ml.SetCursor(-1)
}

func (ml *messagesList) buildItem(index int, cursor int) tview.ScrollListItem {
	if index < 0 || index >= len(ml.messages) {
		return nil
	}

	message := ml.messages[index]
	tv := tview.NewTextView().
		SetWrap(true).
		SetWordWrap(true).
		SetDynamicColors(true).
		SetText(ml.renderMessage(message))
	if index == cursor {
		tv.SetTextStyle(tcell.StyleDefault.Reverse(true))
	}
	return tv
}

func (ml *messagesList) renderMessage(message discord.Message) string {
	var b strings.Builder
	ml.writeMessage(&b, message)
	return b.String()
}

func (ml *messagesList) writeMessage(writer io.Writer, message discord.Message) {
	if ml.cfg.HideBlockedUsers {
		isBlocked := ml.chatView.state.UserIsBlocked(message.Author.ID)
		if isBlocked {
			io.WriteString(writer, "[:red:b]Blocked message[:-:-]")
			return
		}
	}

	// reset
	io.WriteString(writer, "[-:-:-]")

	switch message.Type {
	case discord.DefaultMessage:
		if message.Reference != nil && message.Reference.Type == discord.MessageReferenceTypeForward {
			ml.drawForwardedMessage(writer, message)
		} else {
			ml.drawDefaultMessage(writer, message)
		}
	case discord.GuildMemberJoinMessage:
		ml.drawTimestamps(writer, message.Timestamp)
		ml.drawAuthor(writer, message)
		fmt.Fprint(writer, "joined the server.")
	case discord.InlinedReplyMessage:
		ml.drawReplyMessage(writer, message)
	case discord.ChannelPinnedMessage:
		ml.drawPinnedMessage(writer, message)
	default:
		ml.drawTimestamps(writer, message.Timestamp)
		ml.drawAuthor(writer, message)
	}
}

func (ml *messagesList) formatTimestamp(ts discord.Timestamp) string {
	return ts.Time().In(time.Local).Format(ml.cfg.Timestamps.Format)
}

func (ml *messagesList) drawTimestamps(w io.Writer, ts discord.Timestamp) {
	fmt.Fprintf(w, "[::d]%s[::D] ", ml.formatTimestamp(ts))
}

func (ml *messagesList) drawAuthor(w io.Writer, message discord.Message) {
	name := message.Author.DisplayOrUsername()
	foreground := tcell.ColorDefault

	// Webhooks do not have nicknames or roles.
	if message.GuildID.IsValid() && !message.WebhookID.IsValid() {
		member, err := ml.chatView.state.Cabinet.Member(message.GuildID, message.Author.ID)
		if err != nil {
			slog.Error("failed to get member from state", "guild_id", message.GuildID, "member_id", message.Author.ID, "err", err)
		} else {
			if member.Nick != "" {
				name = member.Nick
			}

			color, ok := state.MemberColor(member, func(id discord.RoleID) *discord.Role {
				r, _ := ml.chatView.state.Cabinet.Role(message.GuildID, id)
				return r
			})
			if ok {
				foreground = tcell.NewHexColor(int32(color))
			}
		}
	}

	fmt.Fprintf(w, "[%s::b]%s[-::B] ", foreground, name)
}

func (ml *messagesList) drawContent(w io.Writer, message discord.Message) {
	c := []byte(tview.Escape(message.Content))
	if ml.chatView.cfg.Markdown {
		ast := discordmd.ParseWithMessage(c, *ml.chatView.state.Cabinet, &message, false)
		ml.renderer.Render(w, c, ast)
	} else {
		w.Write(c) // write the content as is
	}
}

func (ml *messagesList) drawSnapshotContent(w io.Writer, message discord.MessageSnapshotMessage) {
	c := []byte(tview.Escape(message.Content))
	// discordmd doesn't support MessageSnapshotMessage, so we just use write it as is. todo?
	w.Write(c)
}

func (ml *messagesList) drawDefaultMessage(w io.Writer, message discord.Message) {
	if ml.cfg.Timestamps.Enabled {
		ml.drawTimestamps(w, message.Timestamp)
	}

	ml.drawAuthor(w, message)
	ml.drawContent(w, message)

	if message.EditedTimestamp.IsValid() {
		io.WriteString(w, " [::d](edited)[::D]")
	}

	for _, a := range message.Attachments {
		fmt.Fprintln(w)

		fg := ml.cfg.Theme.MessagesList.AttachmentStyle.GetForeground()
		bg := ml.cfg.Theme.MessagesList.AttachmentStyle.GetBackground()
		if ml.cfg.ShowAttachmentLinks {
			fmt.Fprintf(w, "[%s:%s]%s:\n%s[-:-]", fg, bg, a.Filename, a.URL)
		} else {
			fmt.Fprintf(w, "[%s:%s]%s[-:-]", fg, bg, a.Filename)
		}
	}
}

func (ml *messagesList) drawForwardedMessage(w io.Writer, message discord.Message) {
	ml.drawTimestamps(w, message.Timestamp)
	ml.drawAuthor(w, message)
	fmt.Fprintf(w, "[::d]%s [::-]", ml.cfg.Theme.MessagesList.ForwardedIndicator)
	ml.drawSnapshotContent(w, message.MessageSnapshots[0].Message)
	fmt.Fprintf(w, " [::d](%s)[-:-:-] ", ml.formatTimestamp(message.MessageSnapshots[0].Message.Timestamp))
}

func (ml *messagesList) drawReplyMessage(w io.Writer, message discord.Message) {
	// reply
	fmt.Fprintf(w, "[::d]%s ", ml.cfg.Theme.MessagesList.ReplyIndicator)
	if m := message.ReferencedMessage; m != nil {
		m.GuildID = message.GuildID
		ml.drawAuthor(w, *m)
		ml.drawContent(w, *m)
	} else {
		io.WriteString(w, "Original message was deleted")
	}

	io.WriteString(w, "\n")
	// main
	ml.drawDefaultMessage(w, message)
}

func (ml *messagesList) drawPinnedMessage(w io.Writer, message discord.Message) {
	fmt.Fprintf(w, "%s pinned a message", message.Author.DisplayOrUsername())
}

func (ml *messagesList) selectedMessage() (*discord.Message, error) {
	if len(ml.messages) == 0 {
		return nil, errors.New("no messages available")
	}

	cursor := ml.Cursor()
	if cursor == -1 || cursor >= len(ml.messages) {
		return nil, errors.New("no message is currently selected")
	}

	return &ml.messages[cursor], nil
}

func (ml *messagesList) onInputCapture(event *tcell.EventKey) *tcell.EventKey {
	switch event.Name() {
	case ml.cfg.Keys.MessagesList.ScrollUp:
		ml.ScrollUp()
		return nil
	case ml.cfg.Keys.MessagesList.ScrollDown:
		ml.ScrollDown()
		return nil
	case ml.cfg.Keys.MessagesList.ScrollTop:
		ml.ScrollToStart()
		return nil
	case ml.cfg.Keys.MessagesList.ScrollBottom:
		ml.ScrollToEnd()
		return nil

	case ml.cfg.Keys.MessagesList.Cancel:
		ml.clearSelection()
		return nil

	case ml.cfg.Keys.MessagesList.SelectPrevious, ml.cfg.Keys.MessagesList.SelectNext, ml.cfg.Keys.MessagesList.SelectFirst, ml.cfg.Keys.MessagesList.SelectLast, ml.cfg.Keys.MessagesList.SelectReply:
		ml._select(event.Name())
		return nil
	case ml.cfg.Keys.MessagesList.YankID:
		ml.yankID()
		return nil
	case ml.cfg.Keys.MessagesList.YankContent:
		ml.yankContent()
		return nil
	case ml.cfg.Keys.MessagesList.YankURL:
		ml.yankURL()
		return nil
	case ml.cfg.Keys.MessagesList.Open:
		ml.open()
		return nil
	case ml.cfg.Keys.MessagesList.Reply:
		ml.reply(false)
		return nil
	case ml.cfg.Keys.MessagesList.ReplyMention:
		ml.reply(true)
		return nil
	case ml.cfg.Keys.MessagesList.Edit:
		ml.edit()
		return nil
	case ml.cfg.Keys.MessagesList.Delete:
		ml.delete()
		return nil
	case ml.cfg.Keys.MessagesList.DeleteConfirm:
		ml.confirmDelete()
		return nil
	}

	return event
}

func (ml *messagesList) _select(name string) {
	messages := ml.messages
	if len(messages) == 0 {
		return
	}

	cursor := ml.Cursor()

	switch name {
	case ml.cfg.Keys.MessagesList.SelectPrevious:
		switch {
		case cursor == -1:
			cursor = len(messages) - 1
		case cursor > 0:
			cursor--
		}
	case ml.cfg.Keys.MessagesList.SelectNext:
		switch {
		case cursor == -1:
			cursor = len(messages) - 1
		case cursor < len(messages)-1:
			cursor++
		}
	case ml.cfg.Keys.MessagesList.SelectFirst:
		cursor = 0
	case ml.cfg.Keys.MessagesList.SelectLast:
		cursor = len(messages) - 1
	case ml.cfg.Keys.MessagesList.SelectReply:
		if cursor == -1 || cursor >= len(messages) {
			return
		}

		if ref := messages[cursor].ReferencedMessage; ref != nil {
			refIdx := slices.IndexFunc(messages, func(m discord.Message) bool {
				return m.ID == ref.ID
			})
			if refIdx != -1 {
				cursor = refIdx
			}
		} else {
			return
		}
	}

	ml.SetCursor(cursor)
}

func (ml *messagesList) yankID() {
	msg, err := ml.selectedMessage()
	if err != nil {
		slog.Error("failed to get selected message", "err", err)
		return
	}

	go clipboard.Write(clipboard.FmtText, []byte(msg.ID.String()))
}

func (ml *messagesList) yankContent() {
	msg, err := ml.selectedMessage()
	if err != nil {
		slog.Error("failed to get selected message", "err", err)
		return
	}

	go clipboard.Write(clipboard.FmtText, []byte(msg.Content))
}

func (ml *messagesList) yankURL() {
	msg, err := ml.selectedMessage()
	if err != nil {
		slog.Error("failed to get selected message", "err", err)
		return
	}

	go clipboard.Write(clipboard.FmtText, []byte(msg.URL()))
}

func (ml *messagesList) open() {
	msg, err := ml.selectedMessage()
	if err != nil {
		slog.Error("failed to get selected message", "err", err)
		return
	}

	var urls []string
	if msg.Content != "" {
		urls = extractURLs(msg.Content)
	}

	if len(urls) == 0 && len(msg.Attachments) == 0 {
		return
	}

	if len(urls)+len(msg.Attachments) == 1 {
		if len(urls) == 1 {
			go ml.openURL(urls[0])
		} else {
			attachment := msg.Attachments[0]
			if strings.HasPrefix(attachment.ContentType, "image/") {
				go ml.openAttachment(msg.Attachments[0])
			} else {
				go ml.openURL(attachment.URL)
			}
		}
	} else {
		ml.showAttachmentsList(urls, msg.Attachments)
	}
}

func extractURLs(content string) []string {
	src := []byte(content)
	node := parser.NewParser(
		parser.WithBlockParsers(discordmd.BlockParsers()...),
		parser.WithInlineParsers(discordmd.InlineParserWithLink()...),
	).Parse(text.NewReader(src))

	var urls []string
	ast.Walk(node, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if entering {
			switch n := n.(type) {
			case *ast.AutoLink:
				urls = append(urls, string(n.URL(src)))
			case *ast.Link:
				urls = append(urls, string(n.Destination))
			}
		}

		return ast.WalkContinue, nil
	})
	return urls
}

func (ml *messagesList) showAttachmentsList(urls []string, attachments []discord.Attachment) {
	list := tview.NewList().
		SetWrapAround(true).
		SetHighlightFullLine(true).
		ShowSecondaryText(false).
		SetDoneFunc(func() {
			ml.chatView.RemovePage(attachmentsListPageName).SwitchToPage(flexPageName)
			ml.chatView.app.SetFocus(ml)
		})
	list.
		SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
			switch event.Name() {
			case ml.cfg.Keys.MessagesList.SelectPrevious:
				return tcell.NewEventKey(tcell.KeyUp, "", tcell.ModNone)
			case ml.cfg.Keys.MessagesList.SelectNext:
				return tcell.NewEventKey(tcell.KeyDown, "", tcell.ModNone)
			case ml.cfg.Keys.MessagesList.SelectFirst:
				return tcell.NewEventKey(tcell.KeyHome, "", tcell.ModNone)
			case ml.cfg.Keys.MessagesList.SelectLast:
				return tcell.NewEventKey(tcell.KeyEnd, "", tcell.ModNone)
			}

			return event
		})
	list.Box = ui.ConfigureBox(list.Box, &ml.cfg.Theme)

	for i, a := range attachments {
		list.AddItem(a.Filename, "", rune('a'+i), func() {
			if strings.HasPrefix(a.ContentType, "image/") {
				go ml.openAttachment(a)
			} else {
				go ml.openURL(a.URL)
			}
		})
	}

	for i, u := range urls {
		list.AddItem(u, "", rune('1'+i), func() {
			go ml.openURL(u)
		})
	}

	ml.chatView.
		AddAndSwitchToPage(attachmentsListPageName, ui.Centered(list, 0, 0), true).
		ShowPage(flexPageName)
}

func (ml *messagesList) openAttachment(attachment discord.Attachment) {
	resp, err := http.Get(attachment.URL)
	if err != nil {
		slog.Error("failed to fetch the attachment", "err", err, "url", attachment.URL)
		return
	}
	defer resp.Body.Close()

	path := filepath.Join(consts.CacheDir(), "attachments")
	if err := os.MkdirAll(path, os.ModePerm); err != nil {
		slog.Error("failed to create attachments dir", "err", err, "path", path)
		return
	}

	path = filepath.Join(path, attachment.Filename)
	file, err := os.Create(path)
	if err != nil {
		slog.Error("failed to create attachment file", "err", err, "path", path)
		return
	}
	defer file.Close()

	if _, err := io.Copy(file, resp.Body); err != nil {
		slog.Error("failed to copy attachment to file", "err", err)
		return
	}

	if err := open.Start(path); err != nil {
		slog.Error("failed to open attachment file", "err", err, "path", path)
		return
	}
}

func (ml *messagesList) openURL(url string) {
	if err := open.Start(url); err != nil {
		slog.Error("failed to open URL", "err", err, "url", url)
	}
}

func (ml *messagesList) reply(mention bool) {
	message, err := ml.selectedMessage()
	if err != nil {
		slog.Error("failed to get selected message", "err", err)
		return
	}

	name := message.Author.DisplayOrUsername()
	if message.GuildID.IsValid() {
		member, err := ml.chatView.state.Cabinet.Member(message.GuildID, message.Author.ID)
		if err != nil {
			slog.Error("failed to get member from state", "guild_id", message.GuildID, "member_id", message.Author.ID, "err", err)
		} else {
			if member.Nick != "" {
				name = member.Nick
			}
		}
	}

	data := ml.chatView.messageInput.sendMessageData
	data.Reference = &discord.MessageReference{MessageID: message.ID}
	data.AllowedMentions = &api.AllowedMentions{RepliedUser: option.False}

	title := "Replying to "
	if mention {
		data.AllowedMentions.RepliedUser = option.True
		title = "[@] " + title
	}

	ml.chatView.messageInput.sendMessageData = data
	ml.chatView.messageInput.SetTitle(title + name)
	ml.chatView.app.SetFocus(ml.chatView.messageInput)
}

func (ml *messagesList) edit() {
	message, err := ml.selectedMessage()
	if err != nil {
		slog.Error("failed to get selected message", "err", err)
		return
	}

	me, err := ml.chatView.state.Cabinet.Me()
	if err != nil {
		slog.Error("failed to get client user (me)", "err", err)
		return
	}

	if message.Author.ID != me.ID {
		slog.Error("failed to edit message; not the author", "channel_id", message.ChannelID, "message_id", message.ID)
		return
	}

	ml.chatView.messageInput.SetTitle("Editing")
	ml.chatView.messageInput.edit = true
	ml.chatView.messageInput.SetText(message.Content, true)
	ml.chatView.app.SetFocus(ml.chatView.messageInput)
}

func (ml *messagesList) confirmDelete() {
	onChoice := func(choice string) {
		if choice == "Yes" {
			ml.delete()
		}
	}

	ml.chatView.showConfirmModal(
		"Are you sure you want to delete this message?",
		[]string{"Yes", "No"},
		onChoice,
	)
}

func (ml *messagesList) delete() {
	msg, err := ml.selectedMessage()
	if err != nil {
		slog.Error("failed to get selected message", "err", err)
		return
	}

	if msg.GuildID.IsValid() {
		me, err := ml.chatView.state.Cabinet.Me()
		if err != nil {
			slog.Error("failed to get client user (me)", "err", err)
			return
		}

		if msg.Author.ID != me.ID && !ml.chatView.state.HasPermissions(msg.ChannelID, discord.PermissionManageMessages) {
			slog.Error("failed to delete message; missing relevant permissions", "channel_id", msg.ChannelID, "message_id", msg.ID)
			return
		}
	}

	selected := ml.chatView.SelectedChannel()
	if selected == nil {
		return
	}

	if err := ml.chatView.state.DeleteMessage(selected.ID, msg.ID, ""); err != nil {
		slog.Error("failed to delete message", "channel_id", selected.ID, "message_id", msg.ID, "err", err)
		return
	}

	if err := ml.chatView.state.MessageRemove(selected.ID, msg.ID); err != nil {
		slog.Error("failed to delete message", "channel_id", selected.ID, "message_id", msg.ID, "err", err)
		return
	}
}

func (ml *messagesList) requestGuildMembers(guildID discord.GuildID, messages []discord.Message) {
	usersToFetch := make([]discord.UserID, 0, len(messages))
	seen := make(map[discord.UserID]struct{}, len(messages))

	for _, message := range messages {
		// Do not fetch member for a webhook message.
		if message.WebhookID.IsValid() {
			continue
		}

		if member, _ := ml.chatView.state.Cabinet.Member(guildID, message.Author.ID); member == nil {
			userID := message.Author.ID
			if _, ok := seen[userID]; !ok {
				seen[userID] = struct{}{}
				usersToFetch = append(usersToFetch, userID)
			}
		}
	}

	if len(usersToFetch) > 0 {
		err := ml.chatView.state.SendGateway(context.TODO(), &gateway.RequestGuildMembersCommand{
			GuildIDs: []discord.GuildID{guildID},
			UserIDs:  usersToFetch,
		})
		if err != nil {
			slog.Error("failed to request guild members", "guild_id", guildID, "err", err)
			return
		}

		ml.setFetchingChunk(true, 0)
		ml.waitForChunkEvent()
	}
}

func (ml *messagesList) setFetchingChunk(value bool, count uint) {
	ml.fetchingMembers.mu.Lock()
	defer ml.fetchingMembers.mu.Unlock()

	if ml.fetchingMembers.value == value {
		return
	}

	ml.fetchingMembers.value = value

	if value {
		ml.fetchingMembers.done = make(chan struct{})
	} else {
		ml.fetchingMembers.count = count
		close(ml.fetchingMembers.done)
	}
}

func (ml *messagesList) waitForChunkEvent() uint {
	ml.fetchingMembers.mu.Lock()
	if !ml.fetchingMembers.value {
		ml.fetchingMembers.mu.Unlock()
		return 0
	}
	ml.fetchingMembers.mu.Unlock()

	<-ml.fetchingMembers.done
	return ml.fetchingMembers.count
}
