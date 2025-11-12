package cmd

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

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
	"github.com/gdamore/tcell/v2"
	"github.com/skratchdot/open-golang/open"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/text"
	"golang.design/x/clipboard"
)

type messagesList struct {
	*ui.MultiLineList
	cfg               *config.Config

	fetchingMembers struct {
		mu    sync.Mutex
		value bool
		count uint
		done  chan struct{}
	}
}

func newMessagesList(cfg *config.Config) *messagesList {
	ml := &messagesList{
		MultiLineList: ui.NewMultiLineList(),
		cfg:      cfg,
	}

	ml.Box = ui.ConfigureBox(ml.Box, &cfg.Theme)
	ml.
		SetTitle("Messages").
		SetInputCapture(ml.onInputCapture)

	markdown.DefaultRenderer.AddOptions(renderer.WithOption("theme", cfg.Theme))
	return ml
}

func (ml *messagesList) reset() {
	ml.
		Clear().
		SetTitle("")
}

func (ml *messagesList) setTitle(channel discord.Channel) {
	title := ui.ChannelToString(channel)
	if topic := channel.Topic; topic != "" {
		title += " - " + topic
	}

	ml.SetTitle(title)
}

func (ml *messagesList) appendMessages(messages []discord.Message) {
	for _, m := range slices.Backward(messages) {
		ml.appendMessage(m)
	}
	ml.DrawItems()
}

func (ml *messagesList) appendMessage(msg discord.Message) {
	var timestamp string 
	if ml.cfg.Timestamps.Enabled {
		timestamp = ml.getTimestamp(msg.Timestamp)
	}
	author := ml.getAuthor(msg)
	if ml.cfg.HideBlockedUsers && discordState.UserIsBlocked(msg.Author.ID) {
		normalMsg := `["` + msg.ID.String() + `"]` + timestamp + "[:red:b]Blocked message[:-:-]" + `[""]`
		highlightedMsg := `["` + msg.ID.String() + `"]` + timestamp + "[:red:b]Blocked message from[:-:-] " + author + `[""]`
		ml.AppendItem(
			normalMsg,
			msg,
			func(li *ui.ListItem, highlighted bool) {
				if highlighted {
					li.Text = highlightedMsg
				} else {
					li.Text = normalMsg
				}
			},
		)
		return
	}

	msgString := strings.Builder{}
	switch msg.Type {
	case discord.DefaultMessage:
		if msg.Reference != nil && msg.Reference.Type == discord.MessageReferenceTypeForward {
			// discordmd doesn't support MessageSnapshotMessage, so we just use write it as is. todo?
			msgString.WriteString(timestamp)
			msgString.WriteString(author)
			msgString.WriteString("[::d]")
			msgString.WriteString(ml.cfg.Theme.MessagesList.ForwardedIndicator)
			msgString.WriteString(" [::D]")
			msgString.WriteString(tview.Escape(msg.Content))
			msgString.WriteRune(rune('('))
			msgString.WriteString(ml.formatTimestamp(msg.MessageSnapshots[0].Message.Timestamp))
			msgString.WriteString(")[-:-:-]")
		} else {
			msgString.WriteString(timestamp)
			msgString.WriteString(author)
			ml.appendDefaultMessage(&msgString, msg)
		}
	case discord.GuildMemberJoinMessage:
		msgString.WriteString(timestamp)
		msgString.WriteString(author)
		msgString.WriteString(" joined the server.")
	case discord.InlinedReplyMessage:
		msgString.WriteString(ml.cfg.Theme.MessagesList.ReplyIndicator)
		msgString.WriteRune(rune(' '))
		if m := msg.ReferencedMessage; m != nil {
			m.GuildID = msg.GuildID
			msgString.WriteString(ml.getAuthor(*m))
			ml.appendContent(&msgString, *m)
		} else {
			msgString.WriteString("Original message was deleted")
		}
		msgString.WriteRune(rune('\n'))
		msgString.WriteString(timestamp)
		msgString.WriteString(author)
		ml.appendDefaultMessage(&msgString, msg)
	case discord.ChannelPinnedMessage:
		msgString.WriteString(timestamp)
		msgString.WriteString(author)
		msgString.WriteString("pinned a message")
	default:
		msgString.WriteString(timestamp)
		msgString.WriteString("Unkown message type from ")
		msgString.WriteString(author)
	}
	ml.AppendItem(msgString.String(), msg, nil)
}

func (ml *messagesList) formatTimestamp(ts discord.Timestamp) string {
	return ts.Time().In(time.Local).Format(ml.cfg.Timestamps.Format)
}

func (ml *messagesList) getTimestamp(ts discord.Timestamp) string {
	return "[::d]" + ml.formatTimestamp(ts) + "[::D] "
}

func (ml *messagesList) getAuthor(message discord.Message) string {
	name := message.Author.DisplayOrUsername()
	foreground := tcell.ColorDefault
	if message.GuildID.IsValid() {
		member, err := discordState.Cabinet.Member(message.GuildID, message.Author.ID)
		if err != nil {
			slog.Error("failed to get member from state", "guild_id", message.GuildID, "member_id", message.Author.ID, "err", err)
		} else {
			if member.Nick != "" {
				name = member.Nick
			}

			color, ok := state.MemberColor(member, func(id discord.RoleID) *discord.Role {
				r, _ := discordState.Cabinet.Role(message.GuildID, id)
				return r
			})
			if ok {
				foreground = tcell.GetColor(color.String())
			}
		}
	}

	return "[" + foreground.String() + "::b]" + name + "[-::B] "
}

func (ml *messagesList) appendContent(w io.Writer, message discord.Message) {
	c := tview.Escape(message.Content)
	if app.cfg.Markdown {
		ast := discordmd.ParseWithMessage([]byte(c), *discordState.Cabinet, &message, false)
		markdown.DefaultRenderer.Render(w, []byte(c), ast)
	} else {
		io.WriteString(w, c)
	}
}

func (ml *messagesList) appendDefaultMessage(w io.Writer, message discord.Message) {
	ml.appendContent(w, message)

	if message.EditedTimestamp.IsValid() {
		io.WriteString(w, " [::d](edited)[::D]")
	}

	for _, a := range message.Attachments {
		fg, bg, _ := ml.cfg.Theme.MessagesList.AttachmentStyle.Decompose()
		io.WriteString(w, "\n[")
		io.WriteString(w, fg.String())
		io.WriteString(w, ":")
		io.WriteString(w, bg.String())
		io.WriteString(w, "]")
		io.WriteString(w, a.Filename)
		if ml.cfg.ShowAttachmentLinks {
			io.WriteString(w, ":\n")
			io.WriteString(w, a.URL)
		}
		io.WriteString(w, "[-:-]")
	}
}

func (ml *messagesList) onInputCapture(event *tcell.EventKey) *tcell.EventKey {
	switch event.Name() {
	case ml.cfg.Keys.MessagesList.Cancel:
		ml.Highlight(-1)

	case ml.cfg.Keys.MessagesList.SelectPrevious, ml.cfg.Keys.MessagesList.SelectNext, ml.cfg.Keys.MessagesList.SelectFirst, ml.cfg.Keys.MessagesList.SelectLast, ml.cfg.Keys.MessagesList.SelectReply:
		ml._select(event.Name())
	case ml.cfg.Keys.MessagesList.YankID:
		ml.yankID()
	case ml.cfg.Keys.MessagesList.YankContent:
		ml.yankContent()
	case ml.cfg.Keys.MessagesList.YankURL:
		ml.yankURL()
	case ml.cfg.Keys.MessagesList.Open:
		ml.open()
	case ml.cfg.Keys.MessagesList.Reply:
		ml.reply(false)
	case ml.cfg.Keys.MessagesList.ReplyMention:
		ml.reply(true)
	case ml.cfg.Keys.MessagesList.Edit:
		ml.edit()
	case ml.cfg.Keys.MessagesList.Delete:
		ml.delete()
	case ml.cfg.Keys.MessagesList.DeleteConfirm:
		ml.confirmDelete()
	}

	return nil
}

func (ml *messagesList) _select(name string) {
	cnt := ml.GetItemCount()
	idx := ml.GetHighlightedIndex()

	switch name {
	case ml.cfg.Keys.MessagesList.SelectPrevious:
		// If no message is currently selected, select the latest message.
		if idx > 0 {
			ml.Highlight(idx-1)
		} else {
			ml.Highlight(cnt-1)
		}
	case ml.cfg.Keys.MessagesList.SelectNext:
		// If no message is currently selected, select the oldest message.
		if idx < cnt-1 {
			ml.Highlight(idx+1)
		} else {
			ml.Highlight(0)
		}
	case ml.cfg.Keys.MessagesList.SelectFirst:
		ml.Highlight(0)
	case ml.cfg.Keys.MessagesList.SelectLast:
		ml.Highlight(cnt-1)
	case ml.cfg.Keys.MessagesList.SelectReply:
		if idx < 0 {
			return
		}

		if ref := ml.GetItem(idx).HoldValue.(discord.Message).ReferencedMessage; ref != nil {
			for i := 0; i < cnt; i++ {
				if ml.GetItem(i).HoldValue.(discord.Message).ID == ref.ID {
					ml.Highlight(i)
					break
				}
			}
		}
	}

	ml.ScrollToHighlight()
}

func (ml *messagesList) yankID() {
	msg := ml.GetHighlightedItem().HoldValue.(discord.Message)
	go clipboard.Write(
		clipboard.FmtText,
		[]byte(msg.ID.String()),
	)
}

func (ml *messagesList) yankContent() {
	msg := ml.GetHighlightedItem().HoldValue.(discord.Message)
	go clipboard.Write(
		clipboard.FmtText,
		[]byte(msg.Content),
	)
}

func (ml *messagesList) yankURL() {
	msg := ml.GetHighlightedItem().HoldValue.(discord.Message)
	go clipboard.Write(
		clipboard.FmtText,
		[]byte(msg.URL()),
	)
}

func (ml *messagesList) open() {
	msg := ml.GetHighlightedItem().HoldValue.(discord.Message)
	var urls []string
	if msg.Content != "" {
		urls = extractURLs(msg.Content)
	}

	if len(urls)+len(msg.Attachments) == 0 {
		return
	}

	if len(urls)+len(msg.Attachments) != 1 {
		ml.showAttachmentsList(urls, msg.Attachments)
		return
	}

	if len(urls) == 1 {
		go ml.openURL(urls[0])
		return
	}

	attachment := msg.Attachments[0]
	if strings.HasPrefix(attachment.ContentType, "image/") {
		go ml.openAttachment(attachment)
	} else {
		go ml.openURL(attachment.URL)
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
			app.pages.RemovePage(attachmentsListPageName).SwitchToPage(flexPageName)
			app.SetFocus(ml)
		})
	list.
		SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
			switch event.Name() {
			case ml.cfg.Keys.MessagesList.SelectPrevious:
				return tcell.NewEventKey(tcell.KeyUp, 0, tcell.ModNone)
			case ml.cfg.Keys.MessagesList.SelectNext:
				return tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModNone)
			case ml.cfg.Keys.MessagesList.SelectFirst:
				return tcell.NewEventKey(tcell.KeyHome, 0, tcell.ModNone)
			case ml.cfg.Keys.MessagesList.SelectLast:
				return tcell.NewEventKey(tcell.KeyEnd, 0, tcell.ModNone)
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

	app.pages.
		AddAndSwitchToPage(attachmentsListPageName, ui.Centered(list, 0, 0), true).
		ShowPage(flexPageName)
}

func (ml *messagesList) openAttachment(attachment discord.Attachment) {
	resp, err := http.Get(attachment.URL)
	if err != nil {
		slog.Error("failed to fetch the attachment", "err", err, "url", attachment.URL)
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
	msg := ml.GetHighlightedItem().HoldValue.(discord.Message)
	name := msg.Author.DisplayOrUsername()
	if msg.GuildID.IsValid() {
		member, err := discordState.Cabinet.Member(msg.GuildID, msg.Author.ID)
		if err != nil {
			slog.Error("failed to get member from state", "guild_id", msg.GuildID, "member_id", msg.Author.ID, "err", err)
		} else {
			if member.Nick != "" {
				name = member.Nick
			}
		}
	}

	data := app.messageInput.sendMessageData
	data.Reference = &discord.MessageReference{MessageID: msg.ID}
	data.AllowedMentions = &api.AllowedMentions{RepliedUser: option.False}

	title := "Replying to "
	if mention {
		data.AllowedMentions.RepliedUser = option.True
		title = "[@] " + title
	}

	app.messageInput.sendMessageData = data
	app.messageInput.addTitle(title + name)
	app.SetFocus(app.messageInput)
}

func (ml *messagesList) edit() {
	msg := ml.GetHighlightedItem().HoldValue.(discord.Message)
	me, err := discordState.Cabinet.Me()
	if err != nil {
		slog.Error("failed to get client user (me)", "err", err)
		return
	}

	if msg.Author.ID != me.ID {
		slog.Error("failed to edit message; not the author", "channel_id", msg.ChannelID, "message_id", msg.ID)
		return
	}

	app.messageInput.SetTitle("Editing")
	app.messageInput.edit = true
	app.messageInput.SetText(msg.Content, true)
	app.SetFocus(app.messageInput)
}

func (ml *messagesList) confirmDelete() {
	onChoice := func(choice string) {
		if choice == "Yes" {
			ml.delete()
		}
	}

	app.showConfirmModal(
		"Are you sure you want to delete this message?",
		[]string{"Yes", "No"},
		onChoice,
	)
}

func (ml *messagesList) delete() {
	msg := ml.GetHighlightedItem().HoldValue.(discord.Message)
	if msg.GuildID.IsValid() {
		me, err := discordState.Cabinet.Me()
		if err != nil {
			slog.Error("failed to get client user (me)", "err", err)
			return
		}

		if msg.Author.ID != me.ID && !discordState.HasPermissions(msg.ChannelID, discord.PermissionManageMessages) {
			slog.Error("failed to delete message; missing relevant permissions", "channel_id", msg.ChannelID, "message_id", msg.ID)
			return
		}
	}

	if err := discordState.DeleteMessage(app.guildsTree.selectedChannelID, msg.ID, ""); err != nil {
		slog.Error("failed to delete message", "channel_id", app.guildsTree.selectedChannelID, "message_id", msg.ID, "err", err)
		return
	}

	ml.Highlight(-1)

	if err := discordState.MessageRemove(app.guildsTree.selectedChannelID, msg.ID); err != nil {
		slog.Error("failed to delete message", "channel_id", app.guildsTree.selectedChannelID, "message_id", msg.ID, "err", err)
		return
	}

	// No need to redraw messages after deletion, onMessageDelete will do
	// its work after the event returns
}

func (ml *messagesList) requestGuildMembers(gID discord.GuildID, ms []discord.Message) {
	var usersToFetch []discord.UserID
	for _, m := range ms {
		if member, _ := discordState.Cabinet.Member(gID, m.Author.ID); member == nil {
			usersToFetch = append(usersToFetch, m.Author.ID)
		}
	}

	if usersToFetch != nil {
		err := discordState.SendGateway(context.TODO(), &gateway.RequestGuildMembersCommand{
			GuildIDs: []discord.GuildID{gID},
			UserIDs:  slices.Compact(usersToFetch),
		})
		if err != nil {
			slog.Error("failed to request guild members", "guild_id", gID, "err", err)
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
