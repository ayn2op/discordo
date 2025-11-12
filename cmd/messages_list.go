package cmd

import (
	"context"
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

	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/discordo/internal/consts"
	"github.com/ayn2op/discordo/internal/list"
	"github.com/ayn2op/discordo/internal/markdown"
	"github.com/ayn2op/discordo/internal/ui"
	"github.com/ayn2op/tview"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/diamondburned/arikawa/v3/state"
	"github.com/diamondburned/ningen/v3/discordmd"
	"github.com/gdamore/tcell/v2"
	"github.com/skratchdot/open-golang/open"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/text"
)

type item struct {
	*tview.TextView
	message discord.Message
}

func newItem(tv *tview.TextView, message discord.Message) *item {
	return &item{TextView: tv, message: message}
}

func (i *item) Height() int {
	n := i.GetWrappedLineCount()
	borders := i.GetBorders()
	if borders.Has(tview.BordersAll) {
		n += 2
	}
	return n
}

type messagesList struct {
	*list.View

	cfg               *config.Config
	selectedMessageID discord.MessageID

	fetchingMembers struct {
		mu    sync.Mutex
		value bool
		count uint
		done  chan struct{}
	}
}

func newMessagesList(cfg *config.Config) *messagesList {
	ml := &messagesList{
		View: list.NewView(),
		cfg:  cfg,
	}

	ml.Box = ui.ConfigureBox(ml.Box, &cfg.Theme)
	ml.
		SetTitle("Messages").
		SetInputCapture(ml.onInputCapture)

	markdown.DefaultRenderer.AddOptions(renderer.WithOption("theme", cfg.Theme))
	return ml
}

func (ml *messagesList) reset() {
	ml.selectedMessageID = 0
	ml.Clear()
	ml.SetTitle("")
}

func (ml *messagesList) setTitle(channel discord.Channel) {
	title := ui.ChannelToString(channel)
	if topic := channel.Topic; topic != "" {
		title += " - " + topic
	}

	ml.SetTitle(title)
}

func (ml *messagesList) drawMessages(messages []discord.Message) {
	for _, m := range slices.Backward(messages) {
		ml.drawMessage(m)
	}
}

func (ml *messagesList) drawMessage(message discord.Message) {
	tv := tview.NewTextView()
	tv.SetDynamicColors(true)

	if ml.cfg.HideBlockedUsers {
		isBlocked := discordState.UserIsBlocked(message.Author.ID)
		if isBlocked {
			io.WriteString(tv, "[:red:b]Blocked message[:-:-]")
			return
		}
	}

	// reset
	io.WriteString(tv, "[-:-:-]")

	switch message.Type {
	case discord.DefaultMessage:
		if message.Reference != nil && message.Reference.Type == discord.MessageReferenceTypeForward {
			ml.drawForwardedMessage(tv, message)
		} else {
			ml.drawDefaultMessage(tv, message)
		}
	case discord.GuildMemberJoinMessage:
		ml.drawTimestamps(tv, message.Timestamp)
		ml.drawAuthor(tv, message)
		fmt.Fprint(tv, "joined the server.")
	case discord.InlinedReplyMessage:
		ml.drawReplyMessage(tv, message)
	case discord.ChannelPinnedMessage:
		ml.drawPinnedMessage(tv, message)
	default:
		ml.drawTimestamps(tv, message.Timestamp)
		ml.drawAuthor(tv, message)
	}

	ml.Append(newItem(tv, message))
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

	fmt.Fprintf(w, "[%s::b]%s[-::B] ", foreground, name)
}

func (ml *messagesList) drawContent(w io.Writer, message discord.Message) {
	c := []byte(tview.Escape(message.Content))
	if app.cfg.Markdown {
		ast := discordmd.ParseWithMessage(c, *discordState.Cabinet, &message, false)
		markdown.DefaultRenderer.Render(w, c, ast)
	} else {
		w.Write(c)
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

	// for _, a := range message.Attachments {
	// 	fmt.Fprintln(ml)
	//
	// 	fg, bg, _ := ml.cfg.Theme.MessagesList.AttachmentStyle.Decompose()
	// 	if ml.cfg.ShowAttachmentLinks {
	// 		fmt.Fprintf(ml, "[%s:%s]%s:\n%s[-:-]", fg, bg, a.Filename, a.URL)
	// 	} else {
	// 		fmt.Fprintf(ml, "[%s:%s]%s[-:-]", fg, bg, a.Filename)
	// 	}
	// }
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

	io.WriteString(w, tview.NewLine)
	// main
	ml.drawDefaultMessage(w, message)
}

func (ml *messagesList) drawPinnedMessage(w io.Writer, message discord.Message) {
	fmt.Fprintf(w, "%s pinned a message", message.Author.DisplayOrUsername())
}

func (ml *messagesList) onInputCapture(event *tcell.EventKey) *tcell.EventKey {
	switch event.Name() {
	case ml.cfg.Keys.MessagesList.Cancel:
		ml.selectedMessageID = 0

	case ml.cfg.Keys.MessagesList.SelectFirst:
		return tcell.NewEventKey(tcell.KeyHome, 0, tcell.ModNone)
	case ml.cfg.Keys.MessagesList.SelectLast:
		return tcell.NewEventKey(tcell.KeyEnd, 0, tcell.ModNone)
	case ml.cfg.Keys.MessagesList.SelectPrevious:
		return tcell.NewEventKey(tcell.KeyUp, 0, tcell.ModNone)
	case ml.cfg.Keys.MessagesList.SelectNext:
		return tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModNone)
	case ml.cfg.Keys.MessagesList.SelectReply:
	}

	return event
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
