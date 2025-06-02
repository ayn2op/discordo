package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"slices"
	"sync"
	"time"

	"github.com/atotto/clipboard"
	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/discordo/internal/markdown"
	"github.com/ayn2op/discordo/internal/ui"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/diamondburned/ningen/v3/discordmd"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/skratchdot/open-golang/open"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/text"
)

type MessagesText struct {
	*tview.TextView
	cfg               *config.Config
	app               *tview.Application
	selectedMessageID discord.MessageID

	fetchingMembers struct {
		mu    sync.Mutex
		value bool
		done  chan struct{}
	}
}

func newMessagesText(app *tview.Application, cfg *config.Config) *MessagesText {
	mt := &MessagesText{
		TextView: tview.NewTextView(),
		cfg:      cfg,
		app:      app,
	}

	mt.Box = ui.NewConfiguredBox(mt.Box, &cfg.Theme)

	t := cfg.Theme
	mt.
		SetDynamicColors(true).
		SetRegions(true).
		SetWordWrap(true).
		ScrollToEnd().
		SetTextColor(tcell.GetColor(t.MessagesText.ContentColor)).
		SetHighlightedFunc(mt.onHighlighted).
		SetChangedFunc(func() {
			app.Draw()
		}).
		SetTitle("Messages").
		SetInputCapture(mt.onInputCapture)

	markdown.DefaultRenderer.AddOptions(
		renderer.WithOption("emojiColor", t.MessagesText.EmojiColor),
		renderer.WithOption("linkColor", t.MessagesText.LinkColor),
		renderer.WithOption("showNicknames", t.MessagesText.ShowNicknames),
	)

	return mt
}

func (mt *MessagesText) drawMsgs(cID discord.ChannelID) {
	ms, err := discordState.Messages(cID, uint(mt.cfg.MessagesLimit))
	if err != nil {
		slog.Error("failed to get messages", "err", err, "channel_id", cID)
		return
	}

	if app.cfg.Theme.MessagesText.ShowNicknames || app.cfg.Theme.MessagesText.ShowUsernameColors {
		if ch, _ := discordState.Cabinet.Channel(cID); ch.GuildID.IsValid() {
			mt.requestGuildMembers(ch.GuildID, ms)
		}
	}

	for _, m := range slices.Backward(ms) {
		app.messagesText.createMsg(m)
	}
}

func (mt *MessagesText) reset() {
	mt.selectedMessageID = 0
	app.messageInput.replyMessageID = 0

	mt.SetTitle("")
	mt.Clear()
	mt.Highlight()
}

// Region tags are square brackets that contain a region ID in double quotes
// https://pkg.go.dev/github.com/rivo/tview#hdr-Regions_and_Highlights
func (mt *MessagesText) startRegion(msgID discord.MessageID) {
	fmt.Fprintf(mt, `["%s"]`, msgID)
}

// Tags with no region ID ([""]) don't start new regions. They can therefore be used to mark the end of a region.
func (mt *MessagesText) endRegion() {
	fmt.Fprint(mt, `[""]`)
}

func (mt *MessagesText) createMsg(msg discord.Message) {
	mt.startRegion(msg.ID)
	defer mt.endRegion()

	if mt.cfg.HideBlockedUsers {
		isBlocked := discordState.UserIsBlocked(msg.Author.ID)
		if isBlocked {
			fmt.Fprintln(mt, "[:red:b]Blocked message[:-:-]")
			return
		}
	}

	// reset
	io.WriteString(mt, "[-:-:-]")

	switch msg.Type {
	case discord.DefaultMessage:
		if msg.Reference != nil && msg.Reference.Type == discord.MessageReferenceTypeForward {
			mt.createForwardedMsg(msg)
		} else {
			mt.createDefaultMsg(msg)
		}
	case discord.InlinedReplyMessage:
		mt.createReplyMsg(msg)

	case discord.ChannelPinnedMessage:
		fmt.Fprint(mt, "["+mt.cfg.Theme.MessagesText.ContentColor+"]"+msg.Author.Username+" pinned a message"+"[-:-:-]")

	default:
		mt.drawTimestamps(msg.Timestamp)
		mt.drawAuthor(msg)
	}

	fmt.Fprintln(mt)
}

func (mt *MessagesText) formatTimestamp(ts discord.Timestamp) string {
	return ts.Time().In(time.Local).Format(mt.cfg.Timestamps.Format)
}

func (mt *MessagesText) drawTimestamps(ts discord.Timestamp) {
	fmt.Fprintf(mt, "[::d]%s[::D] ", mt.formatTimestamp(ts))
}

func (mt *MessagesText) drawAuthor(msg discord.Message) {
	name := mt.authorName(msg.Author, msg.GuildID)
	color := mt.authorColor(msg.Author, msg.GuildID)
	fmt.Fprintf(mt, "[%s]%s[-] ", color, name)
}

func (mt *MessagesText) drawContent(msg discord.Message) {
	c := []byte(tview.Escape(msg.Content))
	ast := discordmd.ParseWithMessage(c, *discordState.Cabinet, &msg, false)
	if app.cfg.MarkdownEnabled {
		markdown.DefaultRenderer.Render(mt, c, ast)
	} else {
		mt.Write(c) // write the content as is
	}
}

func (mt *MessagesText) drawSnapshotContent(msg discord.MessageSnapshotMessage) {
	c := []byte(tview.Escape(msg.Content))
	// discordmd doesn't support MessageSnapshotMessage, so we just use write it as is. todo?
	mt.Write(c)
}

func (mt *MessagesText) createDefaultMsg(msg discord.Message) {
	if mt.cfg.Timestamps.Enabled {
		mt.drawTimestamps(msg.Timestamp)
	}

	mt.drawAuthor(msg)
	mt.drawContent(msg)

	if msg.EditedTimestamp.IsValid() {
		io.WriteString(mt, " [::d](edited)[::D]")
	}

	for _, a := range msg.Attachments {
		fmt.Fprintln(mt)
		if mt.cfg.ShowAttachmentLinks {
			fmt.Fprintf(mt, "[%s][%s]:\n%s[-]", mt.cfg.Theme.MessagesText.AttachmentColor, a.Filename, a.URL)
		} else {
			fmt.Fprintf(mt, "[%s][%s][-]", mt.cfg.Theme.MessagesText.AttachmentColor, a.Filename)
		}
	}
}

func (mt *MessagesText) createReplyMsg(msg discord.Message) {
	// reply
	fmt.Fprintf(mt, "[::d]%s ", mt.cfg.Theme.MessagesText.ReplyIndicator)
	if refMsg := msg.ReferencedMessage; refMsg != nil {
		refMsg.GuildID = msg.GuildID
		mt.drawAuthor(*refMsg)
		mt.drawContent(*refMsg)
	}

	io.WriteString(mt, tview.NewLine)
	// main
	mt.createDefaultMsg(msg)
}

func (mt *MessagesText) authorName(user discord.User, gID discord.GuildID) string {
	name := user.DisplayOrUsername()
	if app.cfg.Theme.MessagesText.ShowNicknames && gID.IsValid() {
		// Use guild nickname if present
		if member, _ := discordState.Cabinet.Member(gID, user.ID); member != nil && member.Nick != "" {
			name = member.Nick
		}
	}

	return name
}

func (mt *MessagesText) createForwardedMsg(msg discord.Message) {
	mt.drawTimestamps(msg.Timestamp)
	mt.drawAuthor(msg)
	fmt.Fprintf(mt, "[::d]%s [::-]", mt.cfg.Theme.MessagesText.ForwardedIndicator)
	mt.drawSnapshotContent(msg.MessageSnapshots[0].Message)
	fmt.Fprintf(mt, " [::d](%s)[-:-:-] ", mt.formatTimestamp(msg.MessageSnapshots[0].Message.Timestamp))
}

func (mt *MessagesText) authorColor(user discord.User, gID discord.GuildID) string {
	color := mt.cfg.Theme.MessagesText.AuthorColor
	if app.cfg.Theme.MessagesText.ShowUsernameColors && gID.IsValid() {
		// Use color from highest role in guild
		if c, ok := discordState.MemberColor(gID, user.ID); ok {
			color = c.String()
		}
	}

	return color
}

func (mt *MessagesText) selectedMsg() (*discord.Message, error) {
	if !mt.selectedMessageID.IsValid() {
		return nil, errors.New("no message is currently selected")
	}

	msg, err := discordState.Cabinet.Message(app.guildsTree.selectedChannelID, mt.selectedMessageID)
	if err != nil {
		return nil, fmt.Errorf("could not retrieve selected message: %w", err)
	}

	return msg, nil
}

func (mt *MessagesText) selectedMsgIndex() (int, error) {
	ms, err := discordState.Cabinet.Messages(app.guildsTree.selectedChannelID)
	if err != nil {
		return -1, err
	}

	for i, m := range ms {
		if m.ID == mt.selectedMessageID {
			return i, nil
		}
	}

	return -1, nil
}

func (mt *MessagesText) onInputCapture(event *tcell.EventKey) *tcell.EventKey {
	switch event.Name() {
	case mt.cfg.Keys.MessagesText.Cancel:
		mt.selectedMessageID = 0
		app.messageInput.replyMessageID = 0
		mt.Highlight()

	case mt.cfg.Keys.MessagesText.SelectPrevious, mt.cfg.Keys.MessagesText.SelectNext, mt.cfg.Keys.MessagesText.SelectFirst, mt.cfg.Keys.MessagesText.SelectLast, mt.cfg.Keys.MessagesText.SelectReply:
		mt._select(event.Name())
	case mt.cfg.Keys.MessagesText.YankID:
		mt.yankID()
	case mt.cfg.Keys.MessagesText.YankContent:
		mt.yankContent()
	case mt.cfg.Keys.MessagesText.YankURL:
		mt.yankURL()
	case mt.cfg.Keys.MessagesText.Open:
		mt.open()
	case mt.cfg.Keys.MessagesText.Reply:
		mt.reply(false)
	case mt.cfg.Keys.MessagesText.ReplyMention:
		mt.reply(true)
	case mt.cfg.Keys.MessagesText.Delete:
		mt.delete()
	}

	return nil
}

func (mt *MessagesText) _select(name string) {
	ms, err := discordState.Cabinet.Messages(app.guildsTree.selectedChannelID)
	if err != nil {
		slog.Error("failed to get messages", "err", err, "channel_id", app.guildsTree.selectedChannelID)
		return
	}

	msgIdx, err := mt.selectedMsgIndex()
	if err != nil {
		slog.Error("failed to get selected message", "err", err)
		return
	}

	switch name {
	case mt.cfg.Keys.MessagesText.SelectPrevious:
		// If no message is currently selected, select the latest message.
		if len(mt.GetHighlights()) == 0 {
			mt.selectedMessageID = ms[0].ID
		} else if msgIdx < len(ms)-1 {
			mt.selectedMessageID = ms[msgIdx+1].ID
		} else {
			return
		}
	case mt.cfg.Keys.MessagesText.SelectNext:
		// If no message is currently selected, select the latest message.
		if len(mt.GetHighlights()) == 0 {
			mt.selectedMessageID = ms[0].ID
		} else if msgIdx > 0 {
			mt.selectedMessageID = ms[msgIdx-1].ID
		} else {
			return
		}
	case mt.cfg.Keys.MessagesText.SelectFirst:
		mt.selectedMessageID = ms[len(ms)-1].ID
	case mt.cfg.Keys.MessagesText.SelectLast:
		mt.selectedMessageID = ms[0].ID
	case mt.cfg.Keys.MessagesText.SelectReply:
		if mt.selectedMessageID == 0 {
			return
		}

		if ref := ms[msgIdx].ReferencedMessage; ref != nil {
			for _, m := range ms {
				if ref.ID == m.ID {
					mt.selectedMessageID = m.ID
				}
			}
		}
	}

	mt.Highlight(mt.selectedMessageID.String())
	mt.ScrollToHighlight()
}

func (mt *MessagesText) onHighlighted(added, removed, remaining []string) {
	if len(added) > 0 {
		id, err := discord.ParseSnowflake(added[0])
		if err != nil {
			slog.Error("Failed to parse region id as int to use as message id.", "err", err)
			return
		}

		mt.selectedMessageID = discord.MessageID(id)
	}
}

func (mt *MessagesText) yankID() {
	msg, err := mt.selectedMsg()
	if err != nil {
		slog.Error("failed to get selected message", "err", err)
		return
	}

	if err := clipboard.WriteAll(msg.ID.String()); err != nil {
		slog.Error("failed to write to clipboard", "err", err)
	}
}

func (mt *MessagesText) yankContent() {
	msg, err := mt.selectedMsg()
	if err != nil {
		slog.Error("failed to get selected message", "err", err)
		return
	}

	if err = clipboard.WriteAll(msg.Content); err != nil {
		slog.Error("failed to write to clipboard", "err", err)
	}
}

func (mt *MessagesText) yankURL() {
	msg, err := mt.selectedMsg()
	if err != nil {
		slog.Error("failed to get selected message", "err", err)
		return
	}

	if err = clipboard.WriteAll(msg.URL()); err != nil {
		slog.Error("failed to write to clipboard", "err", err)
	}
}

func (mt *MessagesText) open() {
	msg, err := mt.selectedMsg()
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
			go openURL(urls[0])
		} else {
			go openURL(msg.Attachments[0].URL)
		}
	} else {
		mt.showUrlSelector(urls, msg.Attachments)
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

func (mt *MessagesText) showUrlSelector(urls []string, attachments []discord.Attachment) {
	done := func() {
		app.pages.RemovePage("list").SwitchToPage("flex")
		app.SetFocus(app.messagesText)
	}

	list := tview.NewList().
		SetWrapAround(true).
		SetHighlightFullLine(true).
		ShowSecondaryText(false).
		SetDoneFunc(done)

	b := mt.cfg.Theme.Border
	p := b.Padding
	list.
		SetBorder(b.Enabled).
		SetBorderColor(tcell.GetColor(b.Color)).
		SetBorderPadding(p[0], p[1], p[2], p[3])

	list.
		SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
			switch event.Name() {
			case mt.cfg.Keys.MessagesText.SelectPrevious:
				return tcell.NewEventKey(tcell.KeyUp, 0, tcell.ModNone)
			case mt.cfg.Keys.MessagesText.SelectNext:
				return tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModNone)
			case mt.cfg.Keys.MessagesText.SelectFirst:
				return tcell.NewEventKey(tcell.KeyHome, 0, tcell.ModNone)
			case mt.cfg.Keys.MessagesText.SelectLast:
				return tcell.NewEventKey(tcell.KeyEnd, 0, tcell.ModNone)
			}

			return event
		})

	for i, a := range attachments {
		attachment := a
		list.AddItem(a.Filename, "", rune('a'+i), func() {
			go openURL(attachment.URL)
			done()
		})
	}

	for i, u := range urls {
		url := u
		list.AddItem(u, "", rune('1'+i), func() {
			go openURL(url)
			done()
		})
	}

	app.pages.
		AddAndSwitchToPage("list", ui.Centered(list, 0, 0), true).
		ShowPage("flex")
}

func openURL(url string) {
	if err := open.Start(url); err != nil {
		slog.Error("failed to open URL", "err", err, "url", url)
	}
}

func (mt *MessagesText) reply(mention bool) {
	var title string
	if mention {
		title += "[@] Replying to "
	} else {
		title += "Replying to "
	}

	msg, err := mt.selectedMsg()
	if err != nil {
		slog.Error("failed to get selected message", "err", err)
		return
	}

	title += mt.authorName(msg.Author, msg.GuildID)
	app.messageInput.SetTitle(title)
	app.messageInput.replyMessageID = mt.selectedMessageID
	mt.app.SetFocus(app.messageInput)
}

func (mt *MessagesText) delete() {
	msg, err := mt.selectedMsg()
	if err != nil {
		slog.Error("failed to get selected message", "err", err)
		return
	}

	clientID := discordState.Ready().User.ID
	if msg.GuildID.IsValid() {
		ps, err := discordState.Permissions(app.guildsTree.selectedChannelID, discordState.Ready().User.ID)
		if err != nil {
			return
		}

		if msg.Author.ID != clientID && !ps.Has(discord.PermissionManageMessages) {
			return
		}
	} else {
		if msg.Author.ID != clientID {
			return
		}
	}

	if err := discordState.DeleteMessage(app.guildsTree.selectedChannelID, msg.ID, ""); err != nil {
		slog.Error("failed to delete message", "err", err, "channel_id", app.guildsTree.selectedChannelID, "message_id", msg.ID)
		return
	}

	if err := discordState.MessageRemove(app.guildsTree.selectedChannelID, msg.ID); err != nil {
		slog.Error("failed to delete message", "err", err, "channel_id", app.guildsTree.selectedChannelID, "message_id", msg.ID)
		return
	}

	ms, err := discordState.Cabinet.Messages(app.guildsTree.selectedChannelID)
	if err != nil {
		slog.Error("failed to delete message", "err", err, "channel_id", app.guildsTree.selectedChannelID)
		return
	}

	mt.Clear()

	for _, m := range slices.Backward(ms) {
		app.messagesText.createMsg(m)
	}
}

func (mt *MessagesText) requestGuildMembers(gID discord.GuildID, ms []discord.Message) {
	var usersToFetch []discord.UserID
	for _, m := range ms {
		if member, _ := discordState.Cabinet.Member(gID, m.Author.ID); member == nil {
			usersToFetch = append(usersToFetch, m.Author.ID)
		}
	}

	if usersToFetch != nil {
		err := discordState.Gateway().Send(context.Background(), &gateway.RequestGuildMembersCommand{
			GuildIDs: []discord.GuildID{gID},
			UserIDs:  slices.Compact(usersToFetch),
		})
		if err != nil {
			slog.Error("failed to request guild members", "err", err)
			return
		}

		mt.setFetchingChunk(true)
		mt.waitForChunkEvent()
	}
}

func (mt *MessagesText) setFetchingChunk(value bool) {
	mt.fetchingMembers.mu.Lock()
	defer mt.fetchingMembers.mu.Unlock()

	if mt.fetchingMembers.value == value {
		return
	}

	mt.fetchingMembers.value = value

	if value {
		mt.fetchingMembers.done = make(chan struct{})
	} else {
		close(mt.fetchingMembers.done)
	}
}

func (mt *MessagesText) waitForChunkEvent() {
	mt.fetchingMembers.mu.Lock()
	if !mt.fetchingMembers.value {
		mt.fetchingMembers.mu.Unlock()
		return
	}
	mt.fetchingMembers.mu.Unlock()

	select {
	case <-mt.fetchingMembers.done:
	default:
		<-mt.fetchingMembers.done
	}
}
