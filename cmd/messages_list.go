package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
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

func (ml *messagesList) reset() {
	ml.selectedMessageID = 0
	ml.
		Clear().
		Highlight().
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
	for _, m := range slices.Backward(messages) {
		ml.drawMessage(m)
	}
}

func (ml *messagesList) drawMessage(message discord.Message) {
	// Region tags are square brackets that contain a region ID in double quotes
	// https://pkg.go.dev/github.com/ayn2op/tview#hdr-Regions_and_Highlights
	fmt.Fprintf(ml, `["%s"]`, message.ID)
	// Tags with no region ID ([""]) don't start new regions. They can therefore be used to mark the end of a region.
	defer fmt.Fprint(ml, `[""]`)

	if ml.cfg.HideBlockedUsers {
		isBlocked := discordState.UserIsBlocked(message.Author.ID)
		if isBlocked {
			io.WriteString(ml, "[:red:b]Blocked message[:-:-]")
			return
		}
	}

	// reset
	io.WriteString(ml, "[-:-:-]")

	switch message.Type {
	case discord.DefaultMessage:
		if message.Reference != nil && message.Reference.Type == discord.MessageReferenceTypeForward {
			ml.drawForwardedMessage(message)
		} else {
			ml.drawDefaultMessage(message)
		}
	case discord.GuildMemberJoinMessage:
		ml.drawTimestamps(message.Timestamp)
		ml.drawAuthor(message)
		fmt.Fprint(ml, "joined the server.")
	case discord.InlinedReplyMessage:
		ml.drawReplyMessage(message)
	case discord.ChannelPinnedMessage:
		ml.drawPinnedMessage(message)
	default:
		ml.drawTimestamps(message.Timestamp)
		ml.drawAuthor(message)
	}

	fmt.Fprintln(ml)
}

func (ml *messagesList) formatTimestamp(ts discord.Timestamp) string {
	return ts.Time().In(time.Local).Format(ml.cfg.Timestamps.Format)
}

func (ml *messagesList) drawTimestamps(ts discord.Timestamp) {
	fmt.Fprintf(ml, "[::d]%s[::D] ", ml.formatTimestamp(ts))
}

func (ml *messagesList) drawAuthor(message discord.Message) {
	name := message.Author.DisplayOrUsername()
	foreground := tcell.ColorDefault
	if message.GuildID.IsValid() {
		member, err := discordState.Cabinet.Member(message.GuildID, message.Author.ID)
		if err != nil {
			app.onError("Failed to get member from state", err, "guild_id", message.GuildID, "user", message.Author)
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

	fmt.Fprintf(ml, "[%s::b]%s[-::B] ", foreground, name)
}

func (ml *messagesList) drawContent(message discord.Message) {
	c := []byte(tview.Escape(message.Content))
	if app.cfg.Markdown {
		ast := discordmd.ParseWithMessage(c, *discordState.Cabinet, &message, false)
		markdown.DefaultRenderer.Render(ml, c, ast)
	} else {
		ml.Write(c) // write the content as is
	}
}

func (ml *messagesList) drawSnapshotContent(message discord.MessageSnapshotMessage) {
	c := []byte(tview.Escape(message.Content))
	// discordmd doesn't support MessageSnapshotMessage, so we just use write it as is. todo?
	ml.Write(c)
}

func (ml *messagesList) drawDefaultMessage(message discord.Message) {
	if ml.cfg.Timestamps.Enabled {
		ml.drawTimestamps(message.Timestamp)
	}

	ml.drawAuthor(message)
	ml.drawContent(message)

	if message.EditedTimestamp.IsValid() {
		io.WriteString(ml, " [::d](edited)[::D]")
	}

	for _, a := range message.Attachments {
		fmt.Fprintln(ml)

		fg, bg, _ := ml.cfg.Theme.MessagesList.AttachmentStyle.Decompose()
		if ml.cfg.ShowAttachmentLinks {
			fmt.Fprintf(ml, "[%s:%s]%s:\n%s[-:-]", fg, bg, a.Filename, a.URL)
		} else {
			fmt.Fprintf(ml, "[%s:%s]%s[-:-]", fg, bg, a.Filename)
		}
	}
}

func (ml *messagesList) drawForwardedMessage(message discord.Message) {
	ml.drawTimestamps(message.Timestamp)
	ml.drawAuthor(message)
	fmt.Fprintf(ml, "[::d]%s [::-]", ml.cfg.Theme.MessagesList.ForwardedIndicator)
	ml.drawSnapshotContent(message.MessageSnapshots[0].Message)
	fmt.Fprintf(ml, " [::d](%s)[-:-:-] ", ml.formatTimestamp(message.MessageSnapshots[0].Message.Timestamp))
}

func (ml *messagesList) drawReplyMessage(message discord.Message) {
	// reply
	fmt.Fprintf(ml, "[::d]%s ", ml.cfg.Theme.MessagesList.ReplyIndicator)
	if m := message.ReferencedMessage; m != nil {
		m.GuildID = message.GuildID
		ml.drawAuthor(*m)
		ml.drawContent(*m)
	} else {
		io.WriteString(ml, "Original message was deleted")
	}

	io.WriteString(ml, tview.NewLine)
	// main
	ml.drawDefaultMessage(message)
}

func (ml *messagesList) drawPinnedMessage(message discord.Message) {
	fmt.Fprintf(ml, "%s pinned a message", message.Author.DisplayOrUsername())
}

func (ml *messagesList) selectedMessage() (*discord.Message, error) {
	if !ml.selectedMessageID.IsValid() {
		return nil, errors.New("no message is currently selected")
	}

	m, err := discordState.Cabinet.Message(app.guildsTree.selectedChannelID, ml.selectedMessageID)
	if err != nil {
		return nil, fmt.Errorf("could not retrieve selected message: %w", err)
	}

	return m, nil
}

func (ml *messagesList) selectedMessageIndex() (int, error) {
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
	ms, err := discordState.Cabinet.Messages(app.guildsTree.selectedChannelID)
	if err != nil {
		app.onError("Failed to get messages", err, "channel_id", app.guildsTree.selectedChannelID)
		return
	}

	msgIdx, err := ml.selectedMessageIndex()
	if err != nil {
		app.onError("Failed to get selected message", err)
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
			app.onError("Failed to parse region id as int to use as message id", err)
			return
		}

		ml.selectedMessageID = discord.MessageID(id)
	}
}

func (ml *messagesList) yankID() {
	msg, err := ml.selectedMessage()
	if err != nil {
		app.onError("Failed to get selected message", err)
		return
	}

	go clipboard.Write(clipboard.FmtText, []byte(msg.ID.String()))
}

func (ml *messagesList) yankContent() {
	msg, err := ml.selectedMessage()
	if err != nil {
		app.onError("Failed to get selected message", err)
		return
	}

	go clipboard.Write(clipboard.FmtText, []byte(msg.Content))
}

func (ml *messagesList) yankURL() {
	msg, err := ml.selectedMessage()
	if err != nil {
		app.onError("Failed to get selected message", err)
		return
	}

	go clipboard.Write(clipboard.FmtText, []byte(msg.URL()))
}

func (ml *messagesList) open() {
	msg, err := ml.selectedMessage()
	if err != nil {
		app.onError("Failed to get selected message", err)
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
		app.onError("Failed to fetch the attachment", err, "url", attachment.URL)
	}
	defer resp.Body.Close()

	path := filepath.Join(consts.CacheDir(), "attachments")
	if err := os.MkdirAll(path, os.ModePerm); err != nil {
		app.onError("Failed to create attachments dir", err, "path", path)
		return
	}

	path = filepath.Join(path, attachment.Filename)
	file, err := os.Create(path)
	if err != nil {
		app.onError("Failed to create attachment file", err, "path", path)
		return
	}
	defer file.Close()

	if _, err := io.Copy(file, resp.Body); err != nil {
		app.onError("Failed to copy attachment to file", err)
		return
	}

	if err := open.Start(path); err != nil {
		app.onError("Failed to open attachment file", err, "path", path)
		return
	}
}

func (ml *messagesList) openURL(url string) {
	if err := open.Start(url); err != nil {
		app.onError("Failed to open URL", err, "url", url)
	}
}

func (ml *messagesList) reply(mention bool) {
	msg, err := ml.selectedMessage()
	if err != nil {
		app.onError("Failed to get selected message", err)
		return
	}

	name := msg.Author.DisplayOrUsername()
	if msg.GuildID.IsValid() {
		member, err := discordState.Cabinet.Member(msg.GuildID, msg.Author.ID)
		if err != nil {
			app.onError("Failed to get member from state", err, "guild_id", msg.GuildID, "member_id", msg.Author.ID)
		} else {
			if member.Nick != "" {
				name = member.Nick
			}
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

func (ml *messagesList) edit() {
	message, err := ml.selectedMessage()
	if err != nil {
		app.onError("Failed to get selected message", err)
		return
	}

	me, err := discordState.Cabinet.Me()
	if err != nil {
		app.onError("Failed to get client user (me)", err)
		return
	}

	if message.Author.ID != me.ID {
		app.onError("Failed to edit message", errors.New("You are not the author"), "channel_id", message.ChannelID, "message_id", message.ID)
		return
	}

	app.messageInput.SetTitle("Editing")
	app.messageInput.edit = true
	app.messageInput.SetText(message.Content, true)
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
	msg, err := ml.selectedMessage()
	if err != nil {
		app.onError("Failed to get selected message", err)
		return
	}

	if msg.GuildID.IsValid() {
		me, err := discordState.Cabinet.Me()
		if err != nil {
			app.onError("Failed to get client user (me)", err)
			return
		}

		if msg.Author.ID != me.ID && !discordState.HasPermissions(msg.ChannelID, discord.PermissionManageMessages) {
			app.onError("Failed to delete message", errors.New("Permission denied."), "channel_id", msg.ChannelID, "message_id", msg.ID)
			return
		}
	}

	if err := discordState.DeleteMessage(app.guildsTree.selectedChannelID, msg.ID, ""); err != nil {
		app.onError("Failed to delete message", err, "channel_id", app.guildsTree.selectedChannelID, "message_id", msg.ID)
		return
	}

	ml.selectedMessageID = 0
	ml.Highlight()

	if err := discordState.MessageRemove(app.guildsTree.selectedChannelID, msg.ID); err != nil {
		app.onError("Failed to delete message", err, "channel_id", app.guildsTree.selectedChannelID, "message_id", msg.ID)
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
			app.onError("Failed to request guild members", err, "guild_id", gID)
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
