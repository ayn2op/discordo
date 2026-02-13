package chat

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"
	"reflect"
	"unicode/utf8"

	"github.com/ayn2op/tview/layers"

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
	"github.com/gdamore/tcell/v3/color"
	"github.com/skratchdot/open-golang/open"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

type messagesList struct {
	*tview.List
	cfg      *config.Config
	chatView *View
	messages []discord.Message
	// rows is the virtual list model rendered by tview (message rows +
	// date-separator rows). It is rebuilt lazily when rowsDirty is true.
	rows      []messagesListRow
	rowsDirty bool

	renderer *markdown.Renderer
	// itemByID caches unselected message TextViews.
	itemByID map[discord.MessageID]*tview.TextView

	fetchingMembers struct {
		mu    sync.Mutex
		value bool
		count uint
		done  chan struct{}
	}

	hotkeysShowMap map[string]func() bool
}

type messagesListRowKind uint8

const (
	messagesListRowMessage messagesListRowKind = iota
	messagesListRowSeparator
)

type messagesListRow struct {
	kind         messagesListRowKind
	messageIndex int
	timestamp    discord.Timestamp
}

func newMessagesList(cfg *config.Config, chatView *View) *messagesList {
	ml := &messagesList{
		List:     tview.NewList(),
		cfg:      cfg,
		chatView: chatView,
		renderer: markdown.NewRenderer(cfg),
		itemByID: make(map[discord.MessageID]*tview.TextView),
	}

	ml.Box = ui.ConfigureBox(ml.Box, &cfg.Theme)
	ml.SetTitle("Messages")
	ml.SetBuilder(ml.buildItem)
	ml.SetChangedFunc(ml.onRowCursorChanged)
	ml.SetTrackEnd(true)
	ml.SetInputCapture(ml.onInputCapture)
	ml.hotkeysShowMap = map[string]func() bool{
		"goto_reply": ml.hkGotoReply,
		"@/reply": ml.hkReply,
		"edit": ml.hkEdit,
		"delete": ml.hkDelete,
		"open": ml.hkOpen,
	}
	return ml
}

func (ml *messagesList) reset() {
	ml.messages = nil
	ml.rows = nil
	ml.rowsDirty = false
	clear(ml.itemByID)
	ml.
		Clear().
		SetBuilder(ml.buildItem).
		SetTitle("")
}

func (ml *messagesList) setTitle(channel discord.Channel) {
	title := ui.ChannelToString(channel, ml.cfg.Icons)
	if topic := channel.Topic; topic != "" {
		title += " - " + topic
	}

	ml.SetTitle(title)
}

func (ml *messagesList) setMessages(messages []discord.Message) {
	ml.messages = slices.Clone(messages)
	slices.Reverse(ml.messages)
	ml.invalidateRows()
	// New channel payload / refetch: replace the cache wholesale to keep it in
	// lockstep with the current message slice.
	clear(ml.itemByID)
	ml.MarkDirty()
}

func (ml *messagesList) addMessage(message discord.Message) {
	ml.messages = append(ml.messages, message)
	ml.invalidateRows()
	// Defensive invalidation for ID reuse/edits delivered out-of-order.
	delete(ml.itemByID, message.ID)
	ml.MarkDirty()
}

func (ml *messagesList) setMessage(index int, message discord.Message) {
	if index < 0 || index >= len(ml.messages) {
		return
	}

	ml.messages[index] = message
	delete(ml.itemByID, message.ID)
	ml.invalidateRows()
	ml.MarkDirty()
}

func (ml *messagesList) deleteMessage(index int) {
	if index < 0 || index >= len(ml.messages) {
		return
	}

	delete(ml.itemByID, ml.messages[index].ID)
	ml.messages = append(ml.messages[:index], ml.messages[index+1:]...)
	ml.invalidateRows()
	ml.MarkDirty()
}

func (ml *messagesList) clearSelection() {
	ml.SetCursor(-1)
}

func (ml *messagesList) buildItem(index int, cursor int) tview.ListItem {
	ml.ensureRows()

	if index < 0 || index >= len(ml.rows) {
		return nil
	}

	row := ml.rows[index]
	if row.kind == messagesListRowSeparator {
		return ml.buildSeparatorItem(row.timestamp)
	}

	message := ml.messages[row.messageIndex]
	if index == cursor {
		return tview.NewTextView().
			SetWrap(true).
			SetWordWrap(true).
			SetLines(ml.renderMessage(message, ml.cfg.Theme.MessagesList.SelectedMessageStyle.Style))
	}

	item, ok := ml.itemByID[message.ID]
	if !ok {
		item = tview.NewTextView().
			SetWrap(true).
			SetWordWrap(true).
			SetLines(ml.renderMessage(message, ml.cfg.Theme.MessagesList.MessageStyle.Style))
		ml.itemByID[message.ID] = item
	}
	return item
}

func (ml *messagesList) renderMessage(message discord.Message, baseStyle tcell.Style) []tview.Line {
	builder := tview.NewLineBuilder()
	ml.writeMessage(builder, message, baseStyle)
	return builder.Finish()
}

func (ml *messagesList) buildSeparatorItem(ts discord.Timestamp) *tview.TextView {
	builder := tview.NewLineBuilder()
	ml.drawDateSeparator(builder, ts, ml.cfg.Theme.MessagesList.MessageStyle.Style)
	return tview.NewTextView().
		SetScrollable(false).
		SetWrap(false).
		SetWordWrap(false).
		SetLines(builder.Finish())
}

func (ml *messagesList) drawDateSeparator(builder *tview.LineBuilder, ts discord.Timestamp, baseStyle tcell.Style) {
	date := ts.Time().In(time.Local).Format(ml.cfg.DateSeparator.Format)
	label := " " + date + " "
	fillChar := ml.cfg.DateSeparator.Character
	dimStyle := baseStyle.Dim(true)
	_, _, width, _ := ml.GetInnerRect()
	if width <= 0 {
		builder.Write(strings.Repeat(fillChar, 8)+label+strings.Repeat(fillChar, 8), dimStyle)
		return
	}

	labelWidth := utf8.RuneCountInString(label)
	if width <= labelWidth {
		builder.Write(date, dimStyle)
		return
	}

	fillWidth := width - labelWidth
	left := fillWidth / 2
	right := fillWidth - left
	builder.Write(strings.Repeat(fillChar, left)+label+strings.Repeat(fillChar, right), dimStyle)
}

func (ml *messagesList) rebuildRows() {
	rows := make([]messagesListRow, 0, len(ml.messages)*2)

	for i := range ml.messages {
		if ml.cfg.DateSeparator.Enabled && i > 0 && !sameLocalDate(ml.messages[i-1].Timestamp, ml.messages[i].Timestamp) {
			rows = append(rows, messagesListRow{
				kind:      messagesListRowSeparator,
				timestamp: ml.messages[i].Timestamp,
			})
		}

		rows = append(rows, messagesListRow{
			kind:         messagesListRowMessage,
			messageIndex: i,
		})
	}

	ml.rows = rows
	ml.rowsDirty = false
}

func (ml *messagesList) invalidateRows() {
	ml.rowsDirty = true
}

// ensureRows lazily rebuilds list rows. This avoids repeated O(n) row rebuild
// work when multiple message mutations happen close together.
func (ml *messagesList) ensureRows() {
	if !ml.rowsDirty {
		return
	}

	ml.rebuildRows()
}

func sameLocalDate(a discord.Timestamp, b discord.Timestamp) bool {
	ta := a.Time().In(time.Local)
	tb := b.Time().In(time.Local)
	return ta.Year() == tb.Year() && ta.YearDay() == tb.YearDay()
}

// Cursor returns the selected message index, skipping separator rows.
func (ml *messagesList) Cursor() int {
	ml.ensureRows()
	rowIndex := ml.List.Cursor()
	if rowIndex < 0 || rowIndex >= len(ml.rows) {
		return -1
	}

	row := ml.rows[rowIndex]
	if row.kind != messagesListRowMessage {
		return -1
	}
	return row.messageIndex
}

// SetCursor selects a message index and maps it to the corresponding row.
func (ml *messagesList) SetCursor(index int) {
	ml.List.SetCursor(ml.messageToRowIndex(index))
}

func (ml *messagesList) messageToRowIndex(messageIndex int) int {
	ml.ensureRows()
	if messageIndex < 0 || messageIndex >= len(ml.messages) {
		return -1
	}

	for i, row := range ml.rows {
		if row.kind == messagesListRowMessage && row.messageIndex == messageIndex {
			return i
		}
	}

	return -1
}

func (ml *messagesList) onRowCursorChanged(rowIndex int) {
	ml.ensureRows()
	if rowIndex < 0 || rowIndex >= len(ml.rows) || ml.rows[rowIndex].kind == messagesListRowMessage {
		return
	}

	target := ml.nearestMessageRowIndex(rowIndex)
	ml.List.SetCursor(target)
}

// nearestMessageRowIndex expects rowIndex to be within bounds.
func (ml *messagesList) nearestMessageRowIndex(rowIndex int) int {
	for i := rowIndex - 1; i >= 0; i-- {
		if ml.rows[i].kind == messagesListRowMessage {
			return i
		}
	}
	for i := rowIndex + 1; i < len(ml.rows); i++ {
		if ml.rows[i].kind == messagesListRowMessage {
			return i
		}
	}
	return -1
}

func (ml *messagesList) writeMessage(builder *tview.LineBuilder, message discord.Message, baseStyle tcell.Style) {
	if ml.cfg.HideBlockedUsers {
		isBlocked := ml.chatView.state.UserIsBlocked(message.Author.ID)
		if isBlocked {
			builder.Write("Blocked message", baseStyle.Foreground(color.Red).Bold(true))
			return
		}
	}

	switch message.Type {
	case discord.DefaultMessage:
		if message.Reference != nil && message.Reference.Type == discord.MessageReferenceTypeForward {
			ml.drawForwardedMessage(builder, message, baseStyle)
		} else {
			ml.drawDefaultMessage(builder, message, baseStyle)
		}
	case discord.GuildMemberJoinMessage:
		ml.drawTimestamps(builder, message.Timestamp, baseStyle)
		ml.drawAuthor(builder, message, baseStyle)
		builder.Write("joined the server.", baseStyle)
	case discord.InlinedReplyMessage:
		ml.drawReplyMessage(builder, message, baseStyle)
	case discord.ChannelPinnedMessage:
		ml.drawPinnedMessage(builder, message, baseStyle)
	default:
		ml.drawTimestamps(builder, message.Timestamp, baseStyle)
		ml.drawAuthor(builder, message, baseStyle)
	}
}

func (ml *messagesList) formatTimestamp(ts discord.Timestamp) string {
	return ts.Time().In(time.Local).Format(ml.cfg.Timestamps.Format)
}

func (ml *messagesList) drawTimestamps(builder *tview.LineBuilder, ts discord.Timestamp, baseStyle tcell.Style) {
	dimStyle := baseStyle.Dim(true)
	builder.Write(ml.formatTimestamp(ts)+" ", dimStyle)
}

func (ml *messagesList) drawAuthor(builder *tview.LineBuilder, message discord.Message, baseStyle tcell.Style) {
	name := message.Author.DisplayOrUsername()
	foreground := tcell.ColorDefault

	if member := ml.memberForMessage(message); member != nil {
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

	style := baseStyle.Foreground(foreground).Bold(true)
	builder.Write(name+" ", style)
}

func (ml *messagesList) memberForMessage(message discord.Message) *discord.Member {
	// Webhooks do not have nicknames or roles.
	if !message.GuildID.IsValid() || message.WebhookID.IsValid() {
		return nil
	}

	member, err := ml.chatView.state.Cabinet.Member(message.GuildID, message.Author.ID)
	if err != nil {
		slog.Error("failed to get member from state", "guild_id", message.GuildID, "member_id", message.Author.ID, "err", err)
		return nil
	}
	return member
}

func (ml *messagesList) drawContent(builder *tview.LineBuilder, message discord.Message, baseStyle tcell.Style) {
	c := []byte(message.Content)
	if ml.chatView.cfg.Markdown.Enabled {
		root := discordmd.ParseWithMessage(c, *ml.chatView.state.Cabinet, &message, false)
		lines := ml.renderer.RenderLines(c, root, baseStyle)
		if builder.HasCurrentLine() {
			startsWithCodeBlock := false
			if first := root.FirstChild(); first != nil {
				_, startsWithCodeBlock = first.(*ast.FencedCodeBlock)
			}

			if startsWithCodeBlock {
				// Keep code blocks visually separate from "timestamp + author".
				builder.NewLine()
				for len(lines) > 0 && len(lines[0]) == 0 {
					lines = lines[1:]
				}
			} else {
				for len(lines) > 1 && len(lines[0]) == 0 {
					lines = lines[1:]
				}
			}
		}
		builder.AppendLines(lines)
	} else {
		builder.Write(message.Content, baseStyle)
	}
}

func (ml *messagesList) drawSnapshotContent(builder *tview.LineBuilder, message discord.MessageSnapshotMessage, baseStyle tcell.Style) {
	c := []byte(message.Content)
	// discordmd doesn't support MessageSnapshotMessage, so we just use write it as is. todo?
	builder.Write(string(c), baseStyle)
}

func (ml *messagesList) drawDefaultMessage(builder *tview.LineBuilder, message discord.Message, baseStyle tcell.Style) {
	if ml.cfg.Timestamps.Enabled {
		ml.drawTimestamps(builder, message.Timestamp, baseStyle)
	}

	ml.drawAuthor(builder, message, baseStyle)
	ml.drawContent(builder, message, baseStyle)

	if message.EditedTimestamp.IsValid() {
		dimStyle := baseStyle.Dim(true)
		builder.Write(" (edited)", dimStyle)
	}

	attachmentStyle := ui.MergeStyle(baseStyle, ml.cfg.Theme.MessagesList.AttachmentStyle.Style)
	for _, a := range message.Attachments {
		builder.NewLine()
		if ml.cfg.ShowAttachmentLinks {
			builder.Write(a.Filename+":\n"+a.URL, attachmentStyle)
		} else {
			builder.Write(a.Filename, attachmentStyle)
		}
	}
}

func (ml *messagesList) drawForwardedMessage(builder *tview.LineBuilder, message discord.Message, baseStyle tcell.Style) {
	dimStyle := baseStyle.Dim(true)
	ml.drawTimestamps(builder, message.Timestamp, baseStyle)
	ml.drawAuthor(builder, message, baseStyle)
	builder.Write(ml.cfg.Theme.MessagesList.ForwardedIndicator+" ", dimStyle)
	ml.drawSnapshotContent(builder, message.MessageSnapshots[0].Message, baseStyle)
	builder.Write(" ("+ml.formatTimestamp(message.MessageSnapshots[0].Message.Timestamp)+") ", dimStyle)
}

func (ml *messagesList) drawReplyMessage(builder *tview.LineBuilder, message discord.Message, baseStyle tcell.Style) {
	dimStyle := baseStyle.Dim(true)
	// indicator
	builder.Write(ml.cfg.Theme.MessagesList.ReplyIndicator+" ", dimStyle)

	if m := message.ReferencedMessage; m != nil {
		m.GuildID = message.GuildID
		ml.drawAuthor(builder, *m, dimStyle)
		ml.drawContent(builder, *m, dimStyle)
	} else {
		builder.Write("Original message was deleted", dimStyle)
	}

	builder.NewLine()
	// main
	ml.drawDefaultMessage(builder, message, baseStyle)
}

func (ml *messagesList) drawPinnedMessage(builder *tview.LineBuilder, message discord.Message, baseStyle tcell.Style) {
	builder.Write(message.Author.DisplayOrUsername(), baseStyle)
	builder.Write(" pinned a message.", baseStyle)
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
	case ml.cfg.Keybinds.MessagesList.ScrollUp:
		ml.ScrollUp()
		return nil
	case ml.cfg.Keybinds.MessagesList.ScrollDown:
		ml.ScrollDown()
		return nil
	case ml.cfg.Keybinds.MessagesList.ScrollTop:
		ml.ScrollToStart()
		return nil
	case ml.cfg.Keybinds.MessagesList.ScrollBottom:
		ml.ScrollToEnd()
		return nil

	case ml.cfg.Keybinds.MessagesList.Cancel:
		if ml.Cursor() != -1 {
			ml.clearSelection()
		} else {
			ml.chatView.app.SetFocus(ml.chatView.hotkeysBar)
		}
		return nil

	case ml.cfg.Keybinds.MessagesList.SelectUp, ml.cfg.Keybinds.MessagesList.SelectDown, ml.cfg.Keybinds.MessagesList.SelectTop, ml.cfg.Keybinds.MessagesList.SelectBottom, ml.cfg.Keybinds.MessagesList.SelectReply:
		ml._select(event.Name())
		return nil
	case ml.cfg.Keybinds.MessagesList.YankID:
		ml.yankID()
		return nil
	case ml.cfg.Keybinds.MessagesList.YankContent:
		ml.yankContent()
		return nil
	case ml.cfg.Keybinds.MessagesList.YankURL:
		ml.yankURL()
		return nil
	case ml.cfg.Keybinds.MessagesList.Open:
		ml.open()
		return nil
	case ml.cfg.Keybinds.MessagesList.Reply:
		ml.reply(false)
		return nil
	case ml.cfg.Keybinds.MessagesList.ReplyMention:
		ml.reply(true)
		return nil
	case ml.cfg.Keybinds.MessagesList.Edit:
		ml.edit()
		return nil
	case ml.cfg.Keybinds.MessagesList.Delete:
		ml.delete()
		return nil
	case ml.cfg.Keybinds.MessagesList.DeleteConfirm:
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
	case ml.cfg.Keybinds.MessagesList.SelectUp:
		switch {
		case cursor == -1:
			cursor = len(messages) - 1
		case cursor > 0:
			cursor--
		case cursor == 0:
			added := ml.prependOlderMessages()
			if added == 0 {
				return
			}
			cursor = added - 1
		}
	case ml.cfg.Keybinds.MessagesList.SelectDown:
		switch {
		case cursor == -1:
			cursor = len(messages) - 1
		case cursor < len(messages)-1:
			cursor++
		}
	case ml.cfg.Keybinds.MessagesList.SelectTop:
		cursor = 0
	case ml.cfg.Keybinds.MessagesList.SelectBottom:
		cursor = len(messages) - 1
	case ml.cfg.Keybinds.MessagesList.SelectReply:
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
		}
	}

	ml.SetCursor(cursor)
}

func (ml *messagesList) prependOlderMessages() int {
	selectedChannel := ml.chatView.SelectedChannel()
	if selectedChannel == nil {
		return 0
	}

	channelID := selectedChannel.ID
	before := ml.messages[0].ID
	limit := uint(ml.cfg.MessagesLimit)
	messages, err := ml.chatView.state.MessagesBefore(channelID, before, limit)
	if err != nil {
		slog.Error("failed to fetch older messages", "err", err)
		return 0
	}
	if len(messages) == 0 {
		return 0
	}

	if guildID := selectedChannel.GuildID; guildID.IsValid() {
		ml.requestGuildMembers(guildID, messages)
	}

	older := slices.Clone(messages)
	slices.Reverse(older)

	// Defensive invalidation if Discord returns overlapping windows.
	for _, message := range older {
		delete(ml.itemByID, message.ID)
	}
	ml.messages = slices.Concat(older, ml.messages)
	ml.invalidateRows()
	ml.MarkDirty()
	return len(messages)
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
	type attachmentAction struct {
		label    string
		shortcut rune
		open     func()
	}

	closeList := func() {
		ml.chatView.layers.RemoveLayer(attachmentsListLayerName)
		ml.chatView.app.SetFocus(ml)
	}

	var actions []attachmentAction
	for i, a := range attachments {
		attachment := a
		action := func() {
			if strings.HasPrefix(attachment.ContentType, "image/") {
				go ml.openAttachment(attachment)
			} else {
				go ml.openURL(attachment.URL)
			}
		}
		actions = append(actions, attachmentAction{
			label:    attachment.Filename,
			shortcut: rune('a' + i),
			open:     action,
		})
	}
	for i, u := range urls {
		url := u
		actions = append(actions, attachmentAction{
			label:    url,
			shortcut: rune('1' + i),
			open:     func() { go ml.openURL(url) },
		})
	}

	normalItems := make([]*tview.TextView, len(actions))
	selectedItems := make([]*tview.TextView, len(actions))
	for i, action := range actions {
		normalItems[i] = tview.NewTextView().
			SetScrollable(false).
			SetWrap(false).
			SetWordWrap(false).
			SetLines([]tview.Line{{{Text: action.label, Style: tcell.StyleDefault}}})
		selectedItems[i] = tview.NewTextView().
			SetScrollable(false).
			SetWrap(false).
			SetWordWrap(false).
			SetLines([]tview.Line{{{Text: action.label, Style: tcell.StyleDefault.Reverse(true)}}})
	}

	list := tview.NewList().
		SetSnapToItems(true).
		SetBuilder(func(index int, cursor int) tview.ListItem {
			if index < 0 || index >= len(actions) {
				return nil
			}
			if index == cursor {
				return selectedItems[index]
			}
			return normalItems[index]
		})
	list.Box = ui.ConfigureBox(list.Box, &ml.cfg.Theme)
	if len(actions) > 0 {
		list.SetCursor(0)
	}
	list.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Name() {
		case ml.cfg.Keybinds.MessagesList.SelectUp:
			return tcell.NewEventKey(tcell.KeyUp, "", tcell.ModNone)
		case ml.cfg.Keybinds.MessagesList.SelectDown:
			return tcell.NewEventKey(tcell.KeyDown, "", tcell.ModNone)
		case ml.cfg.Keybinds.MessagesList.SelectTop:
			return tcell.NewEventKey(tcell.KeyHome, "", tcell.ModNone)
		case ml.cfg.Keybinds.MessagesList.SelectBottom:
			return tcell.NewEventKey(tcell.KeyEnd, "", tcell.ModNone)
		case ml.cfg.Keybinds.MessagesList.Cancel:
			closeList()
			return nil
		}

		if event.Key() == tcell.KeyEnter || event.Key() == tcell.KeyRune && event.Str() == " " {
			index := list.Cursor()
			if index >= 0 && index < len(actions) {
				actions[index].open()
				closeList()
			}
			return nil
		}

		if event.Key() == tcell.KeyRune {
			key := event.Str()
			if key == "" {
				return event
			}
			ch := []rune(key)[0]
			for index, action := range actions {
				if action.shortcut == ch {
					list.SetCursor(index)
					actions[index].open()
					closeList()
					return nil
				}
			}
		}

		return event
	})

	ml.chatView.layers.
		AddLayer(
			ui.Centered(list, 0, 0),
			layers.WithName(attachmentsListLayerName),
			layers.WithResize(true),
			layers.WithVisible(true),
			layers.WithOverlay(),
		).
		SendToFront(attachmentsListLayerName)
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
	if member := ml.memberForMessage(*message); member != nil && member.Nick != "" {
		name = member.Nick
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

	me, _ := ml.chatView.state.Cabinet.Me()
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
		me, _ := ml.chatView.state.Cabinet.Me()
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

// Set hotkeys on focus.
func (ml *messagesList) Focus(delegate func(p tview.Primitive)) {
	ml.hotkeys()
	ml.List.Focus(delegate)
}

// Set hotkeys on mouse focus.
func (ml *messagesList) MouseHandler() func(action tview.MouseAction, event *tcell.EventMouse, setFocus func(p tview.Primitive)) (consumed bool, capture tview.Primitive) {
	return ml.List.WrapMouseHandler(func(action tview.MouseAction, event *tcell.EventMouse, setFocus func(p tview.Primitive)) (consumed bool, capture tview.Primitive) {
		return ml.List.MouseHandler()(action, event, func(p tview.Primitive) {
			if p == ml.List {
				ml.hotkeys()
			}
			setFocus(p)
		})
	})
}

func (ml *messagesList) hotkeys() {
	ml.chatView.hotkeysBar.hotkeysFromValue(
		reflect.ValueOf(ml.cfg.Keybinds.MessagesList),
		ml.hotkeysShowMap,
	)
}

func (ml *messagesList) hkGotoReply() bool {
	msg, err := ml.selectedMessage()
	if err != nil {
		return false
	}
	return msg.ReferencedMessage != nil
}

func (ml *messagesList) hkReply() bool {
	msg, err := ml.selectedMessage()
	if err != nil {
		return false
	}
	me, _ := ml.chatView.state.Cabinet.Me()
	return msg.Author.ID != me.ID
}

func (ml *messagesList) hkEdit() bool {
	msg, err := ml.selectedMessage()
	if err != nil {
		return false
	}
	me, _ := ml.chatView.state.Cabinet.Me()
	return msg.Author.ID == me.ID
}

func (ml *messagesList) hkDelete() bool {
	if ml.hkEdit() {
		return true
	}
	sel := ml.chatView.SelectedChannel()
	return sel != nil && ml.chatView.state.HasPermissions(sel.ID, discord.PermissionManageMessages)
}

func (ml *messagesList) hkOpen() bool {
	msg, err := ml.selectedMessage()
	if err != nil {
		return false
	}
	return len(extractURLs(msg.Content)) != 0 || len(msg.Attachments) != 0
}
