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
)

type messagesList struct {
	*tview.TextView
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
		TextView: tview.NewTextView(),
		cfg:      cfg,
	}

	ml.Box = ui.ConfigureBox(ml.Box, &cfg.Theme)
	ml.
		SetDynamicColors(true).
		SetRegions(true).
		SetWordWrap(true).
		ScrollToEnd().
		SetHighlightedFunc(ml.onHighlighted).
		SetTitle("Messages").
		SetInputCapture(ml.onInputCapture)

	markdown.DefaultRenderer.AddOptions(renderer.WithOption("theme", cfg.Theme))
	return ml
}

func (ml *messagesList) drawMsgs(cID discord.ChannelID) {
	msgs, err := discordState.Messages(cID, uint(ml.cfg.MessagesLimit))
	if err != nil {
		slog.Error("failed to get messages", "err", err, "channel_id", cID)
		return
	}

	channel, err := discordState.Cabinet.Channel(cID)
	if err != nil {
		slog.Error("failed to get channel from state", "channel_id", cID, "err", err)
		return
	}

	if guildID := channel.GuildID; guildID.IsValid() {
		ml.requestGuildMembers(guildID, msgs)
	}

	for _, m := range slices.Backward(msgs) {
		ml.createMsg(m)
	}
}

func (ml *messagesList) reset() {
	ml.selectedMessageID = 0
	ml.
		Clear().
		Highlight().
		SetTitle("")
}

// Region tags are square brackets that contain a region ID in double quotes
// https://pkg.go.dev/github.com/ayn2op/tview#hdr-Regions_and_Highlights
func (ml *messagesList) startRegion(msgID discord.MessageID) {
	fmt.Fprintf(ml, `["%s"]`, msgID)
}

// Tags with no region ID ([""]) don't start new regions. They can therefore be used to mark the end of a region.
func (ml *messagesList) endRegion() {
	fmt.Fprint(ml, `[""]`)
}

func (ml *messagesList) createMsg(msg discord.Message) {
	ml.startRegion(msg.ID)
	defer ml.endRegion()

	if ml.cfg.HideBlockedUsers {
		isBlocked := discordState.UserIsBlocked(msg.Author.ID)
		if isBlocked {
			io.WriteString(ml, "[:red:b]Blocked message[:-:-]")
			return
		}
	}

	// reset
	io.WriteString(ml, "[-:-:-]")

	switch msg.Type {
	case discord.DefaultMessage:
		if msg.Reference != nil && msg.Reference.Type == discord.MessageReferenceTypeForward {
			ml.createForwardedMsg(msg)
		} else {
			ml.createDefaultMsg(msg)
		}
	case discord.InlinedReplyMessage:
		ml.createReplyMsg(msg)
	case discord.ChannelPinnedMessage:
		fmt.Fprintf(ml, "%s pinned a message", msg.Author.DisplayOrUsername())
	default:
		ml.drawTimestamps(msg.Timestamp)
		ml.drawAuthor(msg)
	}

	fmt.Fprintln(ml)
}

func (ml *messagesList) formatTimestamp(ts discord.Timestamp) string {
	return ts.Time().In(time.Local).Format(ml.cfg.Timestamps.Format)
}

func (ml *messagesList) drawTimestamps(ts discord.Timestamp) {
	fmt.Fprintf(ml, "[::d]%s[::D] ", ml.formatTimestamp(ts))
}

func (ml *messagesList) drawAuthor(msg discord.Message) {
	style := ml.cfg.Theme.MessagesList.AuthorStyle
	name := msg.Author.DisplayOrUsername()
	if msg.GuildID.IsValid() {
		member, err := discordState.Cabinet.Member(msg.GuildID, msg.Author.ID)
		if err != nil {
			slog.Error("failed to get member from state", "guild_id", msg.GuildID, "member_id", msg.Author.ID, "err", err)
			return
		}

		if member.Nick != "" {
			name = member.Nick
		}

		color, ok := state.MemberColor(member, func(id discord.RoleID) *discord.Role {
			r, _ := discordState.Cabinet.Role(msg.GuildID, id)
			return r
		})
		if ok {
			c := tcell.GetColor(color.String())
			style = config.NewStyleWrapper(tcell.StyleDefault.Foreground(c))
		}
	}

	fg, bg, _ := style.Decompose()
	_, _ = fmt.Fprintf(ml, "[%s:%s]%s[-] ", fg.String(), bg.String(), name)
}

func (ml *messagesList) drawContent(msg discord.Message) {
	c := []byte(tview.Escape(msg.Content))
	ast := discordmd.ParseWithMessage(c, *discordState.Cabinet, &msg, false)
	if app.cfg.Markdown {
		markdown.DefaultRenderer.Render(ml, c, ast)
	} else {
		ml.Write(c) // write the content as is
	}
}

func (ml *messagesList) drawSnapshotContent(msg discord.MessageSnapshotMessage) {
	c := []byte(tview.Escape(msg.Content))
	// discordmd doesn't support MessageSnapshotMessage, so we just use write it as is. todo?
	ml.Write(c)
}

func (ml *messagesList) createDefaultMsg(msg discord.Message) {
	if ml.cfg.Timestamps.Enabled {
		ml.drawTimestamps(msg.Timestamp)
	}

	ml.drawAuthor(msg)
	ml.drawContent(msg)

	if msg.EditedTimestamp.IsValid() {
		io.WriteString(ml, " [::d](edited)[::D]")
	}

	for _, a := range msg.Attachments {
		fmt.Fprintln(ml)

		fg, bg, _ := ml.cfg.Theme.MessagesList.AttachmentStyle.Decompose()
		if ml.cfg.ShowAttachmentLinks {
			fmt.Fprintf(ml, "[%s:%s]%s:\n%s[-:-]", fg, bg, a.Filename, a.URL)
		} else {
			fmt.Fprintf(ml, "[%s:%s]%s[-:-]", fg, bg, a.Filename)
		}
	}
}

func (ml *messagesList) createReplyMsg(msg discord.Message) {
	// reply
	fmt.Fprintf(ml, "[::d]%s ", ml.cfg.Theme.MessagesList.ReplyIndicator)
	if refMsg := msg.ReferencedMessage; refMsg != nil {
		refMsg.GuildID = msg.GuildID
		ml.drawAuthor(*refMsg)
		ml.drawContent(*refMsg)
	}

	io.WriteString(ml, tview.NewLine)
	// main
	ml.createDefaultMsg(msg)
}

func (ml *messagesList) createForwardedMsg(msg discord.Message) {
	ml.drawTimestamps(msg.Timestamp)
	ml.drawAuthor(msg)
	fmt.Fprintf(ml, "[::d]%s [::-]", ml.cfg.Theme.MessagesList.ForwardedIndicator)
	ml.drawSnapshotContent(msg.MessageSnapshots[0].Message)
	fmt.Fprintf(ml, " [::d](%s)[-:-:-] ", ml.formatTimestamp(msg.MessageSnapshots[0].Message.Timestamp))
}

func (ml *messagesList) selectedMsg() (*discord.Message, error) {
	if !ml.selectedMessageID.IsValid() {
		return nil, errors.New("no message is currently selected")
	}

	msg, err := discordState.Cabinet.Message(app.guildsTree.selectedChannelID, ml.selectedMessageID)
	if err != nil {
		return nil, fmt.Errorf("could not retrieve selected message: %w", err)
	}

	return msg, nil
}

func (ml *messagesList) selectedMsgIndex() (int, error) {
	ms, err := discordState.Cabinet.Messages(app.guildsTree.selectedChannelID)
	if err != nil {
		return -1, err
	}

	for i, m := range ms {
		if m.ID == ml.selectedMessageID {
			return i, nil
		}
	}

	return -1, nil
}

func (ml *messagesList) onInputCapture(event *tcell.EventKey) *tcell.EventKey {
	switch event.Name() {
	case ml.cfg.Keys.MessagesList.Cancel:
		ml.selectedMessageID = 0
		ml.Highlight()

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
	case ml.cfg.Keys.MessagesList.Delete:
		ml.delete()
	}

	return nil
}

func (ml *messagesList) _select(name string) {
	ms, err := discordState.Cabinet.Messages(app.guildsTree.selectedChannelID)
	if err != nil {
		slog.Error("failed to get messages", "err", err, "channel_id", app.guildsTree.selectedChannelID)
		return
	}

	msgIdx, err := ml.selectedMsgIndex()
	if err != nil {
		slog.Error("failed to get selected message", "err", err)
		return
	}

	switch name {
	case ml.cfg.Keys.MessagesList.SelectPrevious:
		// If no message is currently selected, select the latest message.
		if len(ml.GetHighlights()) == 0 {
			ml.selectedMessageID = ms[0].ID
		} else if msgIdx < len(ms)-1 {
			ml.selectedMessageID = ms[msgIdx+1].ID
		} else {
			return
		}
	case ml.cfg.Keys.MessagesList.SelectNext:
		// If no message is currently selected, select the latest message.
		if len(ml.GetHighlights()) == 0 {
			ml.selectedMessageID = ms[0].ID
		} else if msgIdx > 0 {
			ml.selectedMessageID = ms[msgIdx-1].ID
		} else {
			return
		}
	case ml.cfg.Keys.MessagesList.SelectFirst:
		ml.selectedMessageID = ms[len(ms)-1].ID
	case ml.cfg.Keys.MessagesList.SelectLast:
		ml.selectedMessageID = ms[0].ID
	case ml.cfg.Keys.MessagesList.SelectReply:
		if ml.selectedMessageID == 0 {
			return
		}

		if ref := ms[msgIdx].ReferencedMessage; ref != nil {
			for _, m := range ms {
				if ref.ID == m.ID {
					ml.selectedMessageID = m.ID
				}
			}
		}
	}

	ml.Highlight(ml.selectedMessageID.String())
	ml.ScrollToHighlight()
}

func (ml *messagesList) onHighlighted(added, removed, remaining []string) {
	if len(added) > 0 {
		id, err := discord.ParseSnowflake(added[0])
		if err != nil {
			slog.Error("Failed to parse region id as int to use as message id.", "err", err)
			return
		}

		ml.selectedMessageID = discord.MessageID(id)
	}
}

func (ml *messagesList) yankID() {
	msg, err := ml.selectedMsg()
	if err != nil {
		slog.Error("failed to get selected message", "err", err)
		return
	}

	if err := clipboard.WriteAll(msg.ID.String()); err != nil {
		slog.Error("failed to write to clipboard", "err", err)
	}
}

func (ml *messagesList) yankContent() {
	msg, err := ml.selectedMsg()
	if err != nil {
		slog.Error("failed to get selected message", "err", err)
		return
	}

	if err = clipboard.WriteAll(msg.Content); err != nil {
		slog.Error("failed to write to clipboard", "err", err)
	}
}

func (ml *messagesList) yankURL() {
	msg, err := ml.selectedMsg()
	if err != nil {
		slog.Error("failed to get selected message", "err", err)
		return
	}

	if err = clipboard.WriteAll(msg.URL()); err != nil {
		slog.Error("failed to write to clipboard", "err", err)
	}
}

func (ml *messagesList) open() {
	msg, err := ml.selectedMsg()
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
			go openURL(a.URL)
		})
	}

	for i, u := range urls {
		list.AddItem(u, "", rune('1'+i), func() {
			go openURL(u)
		})
	}

	app.pages.
		AddAndSwitchToPage(attachmentsListPageName, ui.Centered(list, 0, 0), true).
		ShowPage(flexPageName)
}

func openURL(url string) {
	if err := open.Start(url); err != nil {
		slog.Error("failed to open URL", "err", err, "url", url)
	}
}

func (ml *messagesList) reply(mention bool) {
	msg, err := ml.selectedMsg()
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

		if member.Nick != "" {
			name = member.Nick
		}
	}

	data := app.messageInput.sendMessageData
	data.Reference = &discord.MessageReference{MessageID: ml.selectedMessageID}
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

func (ml *messagesList) delete() {
	msg, err := ml.selectedMsg()
	if err != nil {
		slog.Error("failed to get selected message", "err", err)
		return
	}

	clientID := discordState.Ready().User.ID
	if msg.GuildID.IsValid() {
		perms, err := discordState.Permissions(app.guildsTree.selectedChannelID, clientID)
		if err != nil {
			return
		}

		if msg.Author.ID != clientID && !perms.Has(discord.PermissionManageMessages) {
			return
		}
	}

	if err := discordState.DeleteMessage(app.guildsTree.selectedChannelID, msg.ID, ""); err != nil {
		slog.Error("failed to delete message", "channel_id", app.guildsTree.selectedChannelID, "message_id", msg.ID, "err", err)
		return
	}

	ml.selectedMessageID = 0
	ml.Highlight()

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
