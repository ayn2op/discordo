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

type messagesText struct {
	*tview.TextView
	cfg               *config.Config
	selectedMessageID discord.MessageID

	fetchingMembers struct {
		mu    sync.Mutex
		value bool
		count uint
		done  chan struct{}
	}

	urlListPage int
}

func newMessagesText(cfg *config.Config) *messagesText {
	mt := &messagesText{
		TextView: tview.NewTextView(),
		cfg:      cfg,
	}

	mt.Box = ui.NewConfiguredBox(mt.Box, &cfg.Theme)
	mt.
		SetDynamicColors(true).
		SetRegions(true).
		SetWordWrap(true).
		ScrollToEnd().
		SetHighlightedFunc(mt.onHighlighted).
		SetTitle("Messages").
		SetInputCapture(mt.onInputCapture)

	markdown.DefaultRenderer.AddOptions(renderer.WithOption("theme", cfg.Theme.MessagesText))
	return mt
}

func (mt *messagesText) drawMsgs(cID discord.ChannelID) {
	msgs, err := discordState.Messages(cID, uint(mt.cfg.MessagesLimit))
	if err != nil {
		slog.Error("failed to get messages", "err", err, "channel_id", cID)
		return
	}

	if app.cfg.Theme.MessagesText.ShowNicknames || app.cfg.Theme.MessagesText.ShowUsernameColors {
		if ch, _ := discordState.Cabinet.Channel(cID); ch.GuildID.IsValid() {
			mt.requestGuildMembers(ch.GuildID, msgs)
		}
	}

	mt.Clear()

	for _, m := range slices.Backward(msgs) {
		mt.createMsg(m)
	}
}

func (mt *messagesText) reset() {
	mt.selectedMessageID = 0
	app.messageInput.replyMessageID = 0

	mt.SetTitle("")
	mt.Clear()
	mt.Highlight()
}

// Region tags are square brackets that contain a region ID in double quotes
// https://pkg.go.dev/github.com/ayn2op/tview#hdr-Regions_and_Highlights
func (mt *messagesText) startRegion(msgID discord.MessageID) {
	fmt.Fprintf(mt, `["%s"]`, msgID)
}

// Tags with no region ID ([""]) don't start new regions. They can therefore be used to mark the end of a region.
func (mt *messagesText) endRegion() {
	fmt.Fprint(mt, `[""]`)
}

func (mt *messagesText) createMsg(msg discord.Message) {
	mt.startRegion(msg.ID)
	defer mt.endRegion()

	if mt.cfg.HideBlockedUsers {
		isBlocked := discordState.UserIsBlocked(msg.Author.ID)
		if isBlocked {
			io.WriteString(mt, "[:red:b]Blocked message[:-:-]")
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
		fmt.Fprintf(mt, "%s pinned a message", msg.Author.DisplayOrUsername())
	default:
		mt.drawTimestamps(msg.Timestamp)
		mt.drawAuthor(msg)
	}

	fmt.Fprintln(mt)
}

func (mt *messagesText) formatTimestamp(ts discord.Timestamp) string {
	return ts.Time().In(time.Local).Format(mt.cfg.Timestamps.Format)
}

func (mt *messagesText) drawTimestamps(ts discord.Timestamp) {
	fmt.Fprintf(mt, "[::d]%s[::D] ", mt.formatTimestamp(ts))
}

func (mt *messagesText) drawAuthor(msg discord.Message) {
	name := msg.Author.DisplayOrUsername()
	style := mt.cfg.Theme.MessagesText.AuthorStyle

	if msg.GuildID.IsValid() {
		member, err := discordState.Cabinet.Member(msg.GuildID, msg.Author.ID)
		if err != nil {
			slog.Error("failed to get member from state", "guild_id", msg.GuildID, "member_id", msg.Author.ID, "err", err)
			return
		}

		if app.cfg.Theme.MessagesText.ShowNicknames && member.Nick != "" {
			name = member.Nick
		}

		if app.cfg.Theme.MessagesText.ShowUsernameColors {
			color, ok := state.MemberColor(member, func(id discord.RoleID) *discord.Role {
				r, _ := discordState.Cabinet.Role(msg.GuildID, id)
				return r
			})
			if ok {
				c := tcell.GetColor(color.String())
				style = config.NewStyleWrapper(tcell.StyleDefault.Foreground(c))
			}
		}
	}

	fg, bg, _ := style.Decompose()
	_, _ = fmt.Fprintf(mt, "[%s:%s]%s[-] ", fg.String(), bg.String(), name)
}

func (mt *messagesText) drawContent(msg discord.Message) {
	c := []byte(tview.Escape(msg.Content))
	ast := discordmd.ParseWithMessage(c, *discordState.Cabinet, &msg, false)
	if app.cfg.Markdown {
		markdown.DefaultRenderer.Render(mt, c, ast)
	} else {
		mt.Write(c) // write the content as is
	}
}

func (mt *messagesText) drawSnapshotContent(msg discord.MessageSnapshotMessage) {
	c := []byte(tview.Escape(msg.Content))
	// discordmd doesn't support MessageSnapshotMessage, so we just use write it as is. todo?
	mt.Write(c)
}

func (mt *messagesText) createDefaultMsg(msg discord.Message) {
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

		fg, bg, _ := mt.cfg.Theme.MessagesText.AttachmentStyle.Decompose()
		if mt.cfg.ShowAttachmentLinks {
			fmt.Fprintf(mt, "[%s:%s][%s]:\n%s[-]", fg, bg, a.Filename, a.URL)
		} else {
			fmt.Fprintf(mt, "[%s:%s][%s][-]", fg, bg, a.Filename)
		}
	}
}

func (mt *messagesText) createReplyMsg(msg discord.Message) {
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

func (mt *messagesText) createForwardedMsg(msg discord.Message) {
	mt.drawTimestamps(msg.Timestamp)
	mt.drawAuthor(msg)
	fmt.Fprintf(mt, "[::d]%s [::-]", mt.cfg.Theme.MessagesText.ForwardedIndicator)
	mt.drawSnapshotContent(msg.MessageSnapshots[0].Message)
	fmt.Fprintf(mt, " [::d](%s)[-:-:-] ", mt.formatTimestamp(msg.MessageSnapshots[0].Message.Timestamp))
}

func (mt *messagesText) selectedMsg() (*discord.Message, error) {
	if !mt.selectedMessageID.IsValid() {
		return nil, errors.New("no message is currently selected")
	}

	msg, err := discordState.Cabinet.Message(app.guildsTree.selectedChannelID, mt.selectedMessageID)
	if err != nil {
		return nil, fmt.Errorf("could not retrieve selected message: %w", err)
	}

	return msg, nil
}

func (mt *messagesText) selectedMsgIndex() (int, error) {
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

func (mt *messagesText) onInputCapture(event *tcell.EventKey) *tcell.EventKey {
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

func (mt *messagesText) _select(name string) {
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

func (mt *messagesText) onHighlighted(added, removed, remaining []string) {
	if len(added) > 0 {
		id, err := discord.ParseSnowflake(added[0])
		if err != nil {
			slog.Error("Failed to parse region id as int to use as message id.", "err", err)
			return
		}

		mt.selectedMessageID = discord.MessageID(id)
	}
}

func (mt *messagesText) yankID() {
	msg, err := mt.selectedMsg()
	if err != nil {
		slog.Error("failed to get selected message", "err", err)
		return
	}

	if err := clipboard.WriteAll(msg.ID.String()); err != nil {
		slog.Error("failed to write to clipboard", "err", err)
	}
}

func (mt *messagesText) yankContent() {
	msg, err := mt.selectedMsg()
	if err != nil {
		slog.Error("failed to get selected message", "err", err)
		return
	}

	if err = clipboard.WriteAll(msg.Content); err != nil {
		slog.Error("failed to write to clipboard", "err", err)
	}
}

func (mt *messagesText) yankURL() {
	msg, err := mt.selectedMsg()
	if err != nil {
		slog.Error("failed to get selected message", "err", err)
		return
	}

	if err = clipboard.WriteAll(msg.URL()); err != nil {
		slog.Error("failed to write to clipboard", "err", err)
	}
}

func (mt *messagesText) open() {
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

func (mt *messagesText) showUrlSelector(urls []string, attachments []discord.Attachment) {
	done := func() {
		app.pages.RemovePage(mt.urlListPage).SwitchToPage(app.flexPage)
		app.SetFocus(mt)
	}

	list := tview.NewList().
		SetWrapAround(true).
		SetHighlightFullLine(true).
		ShowSecondaryText(false).
		SetDoneFunc(done)
	list.Box = ui.NewConfiguredBox(list.Box, &mt.cfg.Theme)

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
		list.AddItem(a.Filename, "", rune('a'+i), func() {
			go openURL(a.URL)
		})
	}

	for i, u := range urls {
		list.AddItem(u, "", rune('1'+i), func() {
			go openURL(u)
		})
	}

	mt.urlListPage = app.pages.AddAndSwitchToPage(ui.Centered(list, 0, 0), true)
	app.pages.ShowPage(app.flexPage)
}

func openURL(url string) {
	if err := open.Start(url); err != nil {
		slog.Error("failed to open URL", "err", err, "url", url)
	}
}

func (mt *messagesText) reply(mention bool) {
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

	name := msg.Author.DisplayOrUsername()

	if msg.GuildID.IsValid() {
		member, err := discordState.Cabinet.Member(msg.GuildID, msg.Author.ID)
		if err != nil {
			slog.Error("failed to get member from state", "guild_id", msg.GuildID, "member_id", msg.Author.ID, "err", err)
			return
		}

		if app.cfg.Theme.MessagesText.ShowNicknames && member.Nick != "" {
			name = member.Nick
		}
	}

	title += name
	app.messageInput.SetTitle(title)
	app.messageInput.replyMessageID = mt.selectedMessageID
	app.SetFocus(app.messageInput)
}

func (mt *messagesText) delete() {
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

	mt.selectedMessageID = 0
	app.messageInput.replyMessageID = 0
	mt.Highlight()

	if err := discordState.MessageRemove(app.guildsTree.selectedChannelID, msg.ID); err != nil {
		slog.Error("failed to delete message", "err", err, "channel_id", app.guildsTree.selectedChannelID, "message_id", msg.ID)
		return
	}

	// No need to redraw messages after deletion, onMessageDelete will do
	// its work after the event returns
}

func (mt *messagesText) requestGuildMembers(gID discord.GuildID, ms []discord.Message) {
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

		mt.setFetchingChunk(true, 0)
		mt.waitForChunkEvent()
	}
}

func (mt *messagesText) setFetchingChunk(value bool, count uint) {
	mt.fetchingMembers.mu.Lock()
	defer mt.fetchingMembers.mu.Unlock()

	if mt.fetchingMembers.value == value {
		return
	}

	mt.fetchingMembers.value = value

	if value {
		mt.fetchingMembers.done = make(chan struct{})
	} else {
		mt.fetchingMembers.count = count
		close(mt.fetchingMembers.done)
	}
}

func (mt *messagesText) waitForChunkEvent() uint {
	mt.fetchingMembers.mu.Lock()
	if !mt.fetchingMembers.value {
		mt.fetchingMembers.mu.Unlock()
		return 0
	}
	mt.fetchingMembers.mu.Unlock()

	<-mt.fetchingMembers.done
	return mt.fetchingMembers.count
}
