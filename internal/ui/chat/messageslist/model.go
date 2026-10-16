package messageslist

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/ayn2op/tview/layout"

	"github.com/ayn2op/arikawa/v3/api"
	"github.com/ayn2op/arikawa/v3/discord"
	md "github.com/ayn2op/arikawa/v3/markdown"
	"github.com/ayn2op/arikawa/v3/state"
	"github.com/ayn2op/arikawa/v3/utils/httputil"
	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/discordo/internal/markdown"
	"github.com/ayn2op/discordo/internal/ui"
	"github.com/ayn2op/discordo/internal/voice"
	"github.com/ayn2op/ningen/v3"
	"github.com/ayn2op/tview"
	"github.com/ayn2op/tview/button"
	"github.com/ayn2op/tview/column"
	tviewimage "github.com/ayn2op/tview/image"
	"github.com/ayn2op/tview/keybind"
	"github.com/ayn2op/tview/list"
	"github.com/ayn2op/tview/richtext"
	"github.com/ayn2op/tview/row"
	"github.com/ayn2op/tview/scrollbar"
	"github.com/ayn2op/tview/slider"
	"github.com/ayn2op/tview/spacer"
	"github.com/ayn2op/tview/text"
	"github.com/ayn2op/tview/textview"
	"github.com/ayn2op/tview/viewport"
	"github.com/gdamore/tcell/v3"
	"github.com/gdamore/tcell/v3/color"
	"github.com/rivo/uniseg"
	"github.com/yuin/goldmark/v2/ast"
)

type messageItem struct {
	message discord.Message
	// text and reply are the message and what it replies to as wrapped, or nil until they are shown.
	text, reply *messageView
	// A separator shows the date of the message after it.
	separator bool
	// grouped reports whether the message is drawn without a header, under that of the message before it.
	grouped bool

	// A preview is blank while loading and empty if loading failed.
	previews map[discord.URL]tviewimage.Widget
}

type Model struct {
	// cursor is the selected item, or -1 for none, and target the item selected last, which stays in view once deselected.
	cursor, target int
	scrollState    viewport.ScrollState

	cfg   *config.Config
	state *ningen.State
	// channel is the selected channel, or nil for none.
	channel *discord.Channel
	items   []messageItem

	renderer *markdown.Renderer
	avatars  map[discord.URL]tviewimage.Widget
	// voicePlayer plays the voice message played last, and voiceStatus is what it last said of it.
	voicePlayer *voice.Player
	voiceStatus voice.PlayerStatus
	// seeking reports whether the waveform of the voice message is dragged, which is sought once it is let go of.
	seeking bool

	// kitty reports whether previews are drawn with kitty's graphics protocol.
	// kittyImages counts the images sent to the terminal.
	kitty       bool
	kittyImages int
	// cellWidth and cellHeight are the size of a terminal cell in pixels, or zero if the terminal does not report it.
	cellWidth, cellHeight uint
}

func NewModel(cfg *config.Config, state *ningen.State) Model {
	ml := Model{
		cursor:      -1,
		target:      -1,
		cfg:         cfg,
		state:       state,
		renderer:    markdown.NewRenderer(cfg),
		avatars:     map[discord.URL]tviewimage.Widget{},
		voicePlayer: voice.NewPlayer(),
	}

	ml.scrollState.ScrollToEnd()
	return ml
}

// View shows the messages in a box titled with the channel and footed with footer, such as who is typing.
func (m Model) View(focused bool, footer string) tview.Widget {
	return ui.Box(m.listView(focused), m.cfg, focused).Title(m.title()).Footer(footer)
}

// title returns the selected channel and its topic, or "Messages" when none is selected.
func (m Model) title() string {
	if m.channel == nil {
		return "Messages"
	}
	title := ui.ChannelToString(*m.channel, m.cfg.Icons, m.state)
	if topic := m.channel.Topic; topic != "" {
		title += " - " + topic
	}
	return title
}

func (m Model) listView(focused bool) viewport.Widget {
	cfg, theme := m.cfg.UI.ScrollBar, m.cfg.Theme.ScrollBar
	bar := scrollbar.New().
		SymbolSet(cfg.SymbolSet.SymbolSet).
		Style(theme.TrackStyle.Style).
		ThumbStyle(theme.ThumbStyle.Style)
	// The model moves the selection itself, so the list is not focused and only takes clicks.
	l := list.New(m.cursor, len(m.items), m.buildItem).
		// A group's messages have no spacing between them.
		Spacing(func(index int) int {
			if m.items[index].grouped {
				return 0
			}
			return int(m.cfg.UI.MessagesList.Spacing)
		}).
		SelectedStyle(m.cfg.Theme.MessagesList.SelectedMessageStyle.Style).
		OnChange(func(cursor int) tview.Msg { return listMsg(cursor) })
	return viewport.New(l, m.scrollState).
		Target(l.Target(m.target)).
		ScrollBar(bar, cfg.Visibility.ScrollBarVisibility).
		// Older messages are added above, so the position is kept from the newest, which also keeps the newest in view.
		AnchorBottom().
		Keybind(ui.ScrollKeybind(m.cfg.Keybinds.MessagesList.ScrollKeybinds)).
		Focused(focused).
		OnChange(func(s *viewport.ScrollState) tview.Msg { return scrollMsg(*s) })
}

// setCursor selects the item at index and scrolls to it if it changed, or selects none if it is negative.
func (m *Model) setCursor(index int) {
	if index >= 0 && index != m.cursor {
		m.scrollState.ScrollToTarget()
		m.target = index
	}

	m.cursor = max(index, -1)
}

// SetChannel shows messages of channel, newest first as Discord sends them, scrolled to the newest, and requests the members who wrote them.
func (m *Model) SetChannel(channel *discord.Channel, messages []discord.Message) tview.Cmd {
	m.channel = channel
	m.reset()
	m.setMessages(messages)
	if channel.GuildID.IsValid() {
		return tview.Batch(m.requestGuildMembers(channel.GuildID, messages), m.loadPreviews())
	}
	return m.loadPreviews()
}

// ShowNewest clears the selection and scrolls to the newest message, which then stays in view as messages arrive until the user scrolls up.
func (m *Model) ShowNewest() {
	m.clearSelection()
	m.scrollState.ScrollToEnd()
}

// UpdateMessage replaces the shown message with the same ID.
func (m *Model) UpdateMessage(message discord.Message) tview.Cmd {
	m.setMessage(m.indexOf(message.ChannelID, message.ID), message)
	return m.loadPreviews()
}

// RefreshMessage shows the cached state of a message, such as after a reaction.
func (m *Model) RefreshMessage(channelID discord.ChannelID, id discord.MessageID) {
	if message, err := m.state.Cabinet.Message(channelID, id); err == nil {
		m.setMessage(m.indexOf(channelID, id), *message)
	}
}

// DeleteMessage removes the shown message with id.
func (m *Model) DeleteMessage(channelID discord.ChannelID, id discord.MessageID) {
	m.deleteMessage(m.indexOf(channelID, id))
}

// indexOf returns the index of the message with id in the channel, or -1 if it is not shown.
func (m Model) indexOf(channelID discord.ChannelID, id discord.MessageID) int {
	if m.channel == nil || m.channel.ID != channelID {
		return -1
	}
	return slices.IndexFunc(m.items, func(item messageItem) bool { return !item.separator && item.message.ID == id })
}

func (m *Model) reset() {
	m.items = nil
	clear(m.avatars)
	m.cursor, m.target = -1, -1
	m.scrollState = viewport.ScrollState{}
	m.scrollState.ScrollToEnd()
}

func (m *Model) setMessages(messages []discord.Message) {
	m.items = make([]messageItem, 0, len(messages))
	for _, message := range slices.Backward(messages) {
		m.items = append(m.items, messageItem{message: message})
	}
	m.rebuildItems()
}

func (m *Model) AddMessage(message discord.Message) tview.Cmd {
	// Appending keeps the items before it as they are, so only the separator of a new day is added rather than all rebuilt.
	var previous *discord.Message
	if last := len(m.items) - 1; last >= 0 {
		previous = &m.items[last].message
	}

	separator := m.shouldSeparate(previous, message)
	if separator {
		m.items = append(m.items, messageItem{separator: true})
	}

	grouped := !separator && m.shouldGroup(previous, message)
	m.items = append(m.items, messageItem{message: message, grouped: grouped})
	return m.loadPreviews()
}

func (m *Model) setMessage(index int, message discord.Message) {
	if index < 0 || index >= len(m.items) || m.items[index].separator {
		return
	}

	m.items[index] = messageItem{message: message, previews: m.items[index].previews}
	m.rebuildItems()
}

func (m *Model) deleteMessage(index int) {
	if index < 0 || index >= len(m.items) || m.items[index].separator {
		return
	}

	cursor := m.cursor
	if cursor == index {
		cursor = m.messageIndex(index, -1)
		if cursor == -1 {
			cursor = m.messageIndex(index, 1)
		}
	}
	if cursor > index {
		cursor--
	}
	m.items = slices.Delete(m.items, index, index+1)
	m.setCursor(cursor)
	m.rebuildItems()
}

func (m *Model) clearSelection() {
	m.setCursor(-1)
}

func (m Model) buildItem(index int) tview.Widget {
	if index < 0 || index >= len(m.items) {
		return nil
	}
	item := &m.items[index]
	if item.separator {
		date := m.items[index+1].message.Timestamp.Time().In(time.Local).Format(m.cfg.DateSeparator.Format)
		return dateSeparator{date: date, fill: m.cfg.DateSeparator.Character, style: m.cfg.Theme.MessagesList.MessageStyle.Dim(true)}
	}

	message := &item.message
	if item.text == nil {
		item.text, item.reply = m.messageViews(*message, item.grouped)
	}

	body := column.New(item.text).Height(layout.Shrink)
	if isVoiceMessage(*message) {
		body = body.Push(m.voiceMessageView(message.Attachments[0]))
	}

	for _, s := range previewSources(*message) {
		if preview, ok := item.previews[s.proxy]; ok {
			body = body.Push(preview)
		}
	}

	if avatar, ok := m.avatars[avatarURL(message.Author)]; ok {
		var gutter tview.Widget = avatar
		// A grouped message leaves the avatar's column empty, under the avatar of the group.
		if item.grouped {
			width, _ := avatar.Size()
			gutter = spacer.New().Width(width)
		}
		body = row.New(gutter, body).Spacing(1).Height(layout.Shrink)
	}

	if message.Type == discord.InlinedReplyMessage && !m.blocked(message.Author) {
		body = column.New(item.reply, body).Height(layout.Shrink)
	}

	return body
}

// messageViews returns the views of the text of message and of what it replies to.
// They capture the model and the message, so they are made here: capturing them in buildItem would copy both to the heap on every call, as the list builds every item each time it is laid out.
func (m Model) messageViews(message discord.Message, grouped bool) (text, reply *messageView) {
	style := m.cfg.Theme.MessagesList.MessageStyle.Style
	text = &messageView{render: func(width int) richtext.Text { return m.renderMessage(message, style, width, grouped) }}
	reply = &messageView{render: func(int) richtext.Text {
		builder := new(richtext.Builder)
		m.drawReply(builder, message, style)
		return builder.Finish()
	}}
	return text, reply
}

func (m Model) blocked(user discord.User) bool {
	return m.cfg.HideBlockedUsers && m.state.UserIsBlocked(user.ID)
}

// messageView is the text of a message. It keeps the wrapped lines until the width changes, as the list measures every message when it draws.
type messageView struct {
	render   func(width int) richtext.Text
	width    int
	rendered bool
	lines    richtext.Text
}

func (v *messageView) at(width int) textview.Widget {
	if !v.rendered || v.width != width {
		v.lines, v.width, v.rendered = nil, width, true
		for _, line := range v.render(width) {
			v.lines = append(v.lines, richtext.WrapWords(line, width)...)
		}
	}
	return textview.New(v.lines).Wrap(false)
}

func (v *messageView) Size() (width, height layout.Length) { return layout.Fill, layout.Shrink }

func (v *messageView) Layout(limits layout.Limits) layout.Size {
	return layout.Sized(limits, layout.Fill, layout.Shrink, func(limits layout.Limits) layout.Size {
		v.at(limits.Max.Width)
		return layout.Size{Height: len(v.lines)}
	})
}

func (v *messageView) Draw(screen tview.Screen, area tview.Rectangle) {
	v.at(area.Width).Draw(screen, area)
}

func (v *messageView) Handle(msg tview.Msg, _ tview.Rectangle) tview.Msg { return msg }

// dateSeparator is a line with the date in the middle, filled to the width it is drawn at.
type dateSeparator struct {
	date, fill string
	style      tview.Style
}

// Size returns Fill and a height of one row.
func (dateSeparator) Size() (width, height layout.Length) { return layout.Fill, layout.Fixed(1) }

// Layout returns a height of one row.
func (dateSeparator) Layout(limits layout.Limits) layout.Size {
	return layout.Atomic(limits, layout.Fill, layout.Fixed(1))
}

func (d dateSeparator) Draw(screen tview.Screen, area tview.Rectangle) {
	line := d.date
	label := " " + d.date + " "
	if labelWidth := utf8.RuneCountInString(label); area.Width > labelWidth {
		fill := area.Width - labelWidth
		line = strings.Repeat(d.fill, fill/2) + label + strings.Repeat(d.fill, fill-fill/2)
	}
	tview.Print(screen, line, area.X, area.Y, area.Width, tview.AlignmentLeft, d.style)
}

func (dateSeparator) Handle(msg tview.Msg, area tview.Rectangle) tview.Msg { return msg }

func (m Model) renderMessage(message discord.Message, baseStyle tview.Style, width int, grouped bool) richtext.Text {
	builder := new(richtext.Builder)
	m.writeMessage(builder, message, baseStyle, width, grouped)
	text := builder.Finish()
	// Without a header or content, the first line is the empty one that attachments and embeds start below.
	if len(text) > 1 && len(text[0]) == 0 {
		text = text[1:]
	}
	return text
}

// shouldGroup reports whether message is drawn under the header of previous, the message before it, which is nil for the first.
func (m Model) shouldGroup(previous *discord.Message, message discord.Message) bool {
	cfg := m.cfg.UI.MessagesList.Grouping
	return cfg.Enabled && previous != nil &&
		// Replies and system messages start a group, as their header says more than who wrote them.
		message.Type == discord.DefaultMessage &&
		(previous.Type == discord.DefaultMessage || previous.Type == discord.InlinedReplyMessage) &&
		// Webhooks post under one ID with a name of each message's own.
		message.Author.ID == previous.Author.ID && message.Author.Username == previous.Author.Username &&
		message.Timestamp.Time().Sub(previous.Timestamp.Time()) <= cfg.Interval
}

// rebuildItems replaces date separators and groups while retaining message views and selection.
func (m *Model) rebuildItems() {
	items := make([]messageItem, 0, len(m.items))
	cursor, target := -1, -1
	var previous *discord.Message
	for i, item := range m.items {
		if item.separator {
			continue
		}

		separator := m.shouldSeparate(previous, item.message)
		if separator {
			items = append(items, messageItem{separator: true})
		}
		// A message that joins or leaves a group is drawn again, with or without its header.
		if grouped := !separator && m.shouldGroup(previous, item.message); grouped != item.grouped {
			item.grouped, item.text = grouped, nil
		}
		if i == m.cursor {
			cursor = len(items)
		}
		if i == m.target {
			target = len(items)
		}
		items = append(items, item)
		previous = &item.message
	}
	m.items = items
	// The selected and target messages only move with the separators, so they are not scrolled to.
	m.cursor, m.target = cursor, target
}

// shouldSeparate reports whether a date separator goes before message, as it is the first (previous is nil) or starts a new day.
func (m Model) shouldSeparate(previous *discord.Message, message discord.Message) bool {
	return m.cfg.DateSeparator.Enabled && (previous == nil || !sameLocalDate(previous.Timestamp, message.Timestamp))
}

func sameLocalDate(a discord.Timestamp, b discord.Timestamp) bool {
	ta := a.Time().In(time.Local)
	tb := b.Time().In(time.Local)
	return ta.Year() == tb.Year() && ta.YearDay() == tb.YearDay()
}

// messageIndex finds the next message in direction (-1 or 1), excluding start.
func (m Model) messageIndex(start, direction int) int {
	for i := start + direction; i >= 0 && i < len(m.items); i += direction {
		if !m.items[i].separator {
			return i
		}
	}
	return -1
}

func (m *Model) onRowCursorChanged(index int) {
	if index < 0 || index >= len(m.items) || !m.items[index].separator {
		return
	}
	target := m.messageIndex(index, -1)
	if target == -1 {
		target = m.messageIndex(index, 1)
	}
	m.setCursor(target)
}

func (m Model) writeMessage(builder *richtext.Builder, message discord.Message, baseStyle tview.Style, width int, grouped bool) {
	if m.blocked(message.Author) {
		builder.Write("Blocked message", baseStyle.Foreground(color.Red).Bold(true))
		return
	}

	switch message.Type {
	case discord.DefaultMessage:
		if message.Reference != nil && message.Reference.Type == discord.MessageReferenceTypeForward {
			m.drawForwardedMessage(builder, message, baseStyle, grouped)
		} else {
			m.drawDefaultMessage(builder, message, baseStyle, width, grouped)
		}
	case discord.GuildMemberJoinMessage:
		m.drawLayout(builder, message, baseStyle, false, func() { builder.Write("joined the server.", baseStyle) })
	case discord.InlinedReplyMessage:
		m.drawDefaultMessage(builder, message, baseStyle, width, false)
	case discord.ChannelPinnedMessage:
		m.drawPinnedMessage(builder, message, baseStyle)
	default:
		m.drawLayout(builder, message, baseStyle, false, nil)
	}
	m.drawReactions(builder, message.Reactions, baseStyle)
}

func (m Model) drawReactions(builder *richtext.Builder, reactions []discord.Reaction, baseStyle tview.Style) {
	if len(reactions) == 0 {
		return
	}

	builder.NewLine()
	for i, reaction := range reactions {
		if i > 0 {
			builder.Write("  ", baseStyle)
		}

		name := reaction.Emoji.Name
		if reaction.Emoji.IsCustom() {
			name = ":" + name + ":"
		}
		style := m.cfg.Theme.MessagesList.ReactionStyle.Style
		if reaction.Me {
			style = m.cfg.Theme.MessagesList.OwnReactionStyle.Style
		}
		builder.Write(name+" "+strconv.Itoa(reaction.Count), tview.MergeStyle(baseStyle, style))
	}
}

func (m Model) formatTimestamp(ts discord.Timestamp) string {
	return ts.Time().In(time.Local).Format(m.cfg.Timestamps.Format)
}

const editedIndicator = " (edited)"

// drawLayout draws the header of message, unless it is grouped, and its content.
func (m Model) drawLayout(builder *richtext.Builder, message discord.Message, baseStyle tview.Style, grouped bool, content func()) {
	if m.cfg.UI.MessagesList.Compact {
		m.drawCompact(builder, message, baseStyle, grouped, content)
		return
	}

	m.drawCozy(builder, message, baseStyle, grouped, content)
}

func (m Model) drawCompact(builder *richtext.Builder, message discord.Message, baseStyle tview.Style, grouped bool, content func()) {
	dimStyle := baseStyle.Dim(true)
	if m.cfg.Timestamps.Enabled {
		builder.Write(m.formatTimestamp(message.Timestamp)+" ", dimStyle)
	}

	if !grouped {
		m.drawAuthor(builder, message, baseStyle)
		if content != nil {
			builder.Write(" ", baseStyle)
		}
	}
	if content != nil {
		content()
	}

	if message.EditedTimestamp.IsValid() {
		builder.Write(editedIndicator, dimStyle)
	}
}

func (m Model) drawCozy(builder *richtext.Builder, message discord.Message, baseStyle tview.Style, grouped bool, content func()) {
	dimStyle := baseStyle.Dim(true)
	if grouped {
		if content != nil {
			content()
		}
		if message.EditedTimestamp.IsValid() {
			builder.Write(editedIndicator, dimStyle)
		}
		return
	}

	m.drawAuthor(builder, message, baseStyle)
	if m.cfg.Timestamps.Enabled {
		builder.Write(" "+m.formatTimestamp(message.Timestamp), dimStyle)
	}

	if message.EditedTimestamp.IsValid() {
		builder.Write(editedIndicator, dimStyle)
	}

	if content != nil {
		builder.NewLine()
		content()
	}
}

func (m Model) drawAuthor(builder *richtext.Builder, message discord.Message, baseStyle tview.Style) {
	name := message.Author.DisplayOrUsername()
	foreground := tcell.ColorDefault

	if member := m.memberForMessage(message); member != nil {
		if member.Nick != "" {
			name = member.Nick
		}

		color, ok := state.MemberColor(member, func(id discord.RoleID) *discord.Role {
			r, _ := m.state.Cabinet.Role(message.GuildID, id)
			return r
		})
		if ok {
			foreground = tcell.NewHexColor(int32(color))
		}
	}

	style := baseStyle.Foreground(foreground).Bold(true)
	builder.Write(name, style)
}

func (m Model) memberForMessage(message discord.Message) *discord.Member {
	// Webhooks do not have nicknames or roles.
	if !message.GuildID.IsValid() || message.WebhookID.IsValid() {
		return nil
	}

	member, err := m.state.Cabinet.Member(message.GuildID, message.Author.ID)
	if err != nil {
		slog.Error("failed to get member from state", "guild_id", message.GuildID, "member_id", message.Author.ID, "err", err)
		return nil
	}
	return member
}

// drawContent renders the message body and returns the parsed markdown AST together with the source bytes it indexes into, so callers can reuse them instead of re-parsing the same content (see drawEmbeds).
// root is nil when markdown rendering is disabled.
func (m Model) drawContent(builder *richtext.Builder, message discord.Message, baseStyle tview.Style) (ast.Node, []byte) {
	content, root, source := m.renderContent(message, baseStyle, false)
	if m.cfg.Markdown.Enabled {
		startsWithCodeBlock := false
		if root != nil {
			if first := root.FirstChild(); first != nil {
				_, startsWithCodeBlock = first.(*ast.CodeBlock)
			}
		}

		if startsWithCodeBlock {
			// Keep code blocks visually separate from "timestamp + author".
			if !builder.LineEmpty() {
				builder.NewLine()
			}

			for len(content) > 0 && len(content[0]) == 0 {
				content = content[1:]
			}
		} else {
			for len(content) > 1 && len(content[0]) == 0 {
				content = content[1:]
			}
		}
	}
	builder.WriteText(content)
	return root, source
}

func (m Model) renderContent(message discord.Message, baseStyle tview.Style, forceMarkdown bool) (richtext.Text, ast.Node, []byte) {
	// Keep one rendering path for both normal messages and embed fragments so we preserve mention/link parsing behavior consistently across both.
	if forceMarkdown || m.cfg.Markdown.Enabled {
		c := []byte(message.Content)
		root := md.ParseWithMessage(c, *m.state.Cabinet, &message)
		return m.renderer.RenderText(c, root, baseStyle), root, c
	}

	b := new(richtext.Builder)
	b.Write(message.Content, baseStyle)
	return b.Finish(), nil, nil
}

func (m Model) drawSnapshotContent(builder *richtext.Builder, parent discord.Message, snapshot discord.MessageSnapshotMessage, baseStyle tview.Style) {
	// Convert discord.MessageSnapshotMessage to discord.Message with common fields.
	message := discord.Message{
		Type:            snapshot.Type,
		Content:         snapshot.Content,
		Embeds:          snapshot.Embeds,
		Attachments:     snapshot.Attachments,
		Timestamp:       snapshot.Timestamp,
		EditedTimestamp: snapshot.EditedTimestamp,
		Flags:           snapshot.Flags,
		Mentions:        snapshot.Mentions,
		MentionRoleIDs:  snapshot.MentionRoleIDs,
		Stickers:        snapshot.Stickers,
		Components:      snapshot.Components,
		ChannelID:       parent.ChannelID,
		GuildID:         parent.GuildID,
	}
	m.drawContent(builder, message, baseStyle)
}

func (m Model) drawDefaultMessage(builder *richtext.Builder, message discord.Message, baseStyle tview.Style, width int, grouped bool) {
	var contentRoot ast.Node
	var contentSource []byte
	content := func() { contentRoot, contentSource = m.drawContent(builder, message, baseStyle) }
	if message.Content == "" {
		content = nil
	}

	m.drawLayout(builder, message, baseStyle, grouped, content)
	m.drawEmbeds(builder, message, baseStyle, width, contentRoot, contentSource)

	attachmentStyle := tview.MergeStyle(baseStyle, m.cfg.Theme.MessagesList.AttachmentStyle.Style)
	// A voice message is shown as its player.
	if isVoiceMessage(message) {
		return
	}

	for _, a := range message.Attachments {
		builder.NewLine()
		if m.cfg.UI.MessagesList.Attachments.ShowLinks {
			builder.Write(a.Filename+":", attachmentStyle)
			builder.NewLine()
			builder.Write(a.URL, attachmentStyle.Url(a.URL))
		} else {
			builder.Write(a.Filename, attachmentStyle)
		}
	}
}

const (
	symbolPlay  = "▶"
	symbolPause = "⏸"
	// positionSeparator is between how much of a voice message was played and how long it is.
	positionSeparator = " / "
	// waveformBars is what the waveform of a voice message is drawn with, from quiet to loud.
	waveformBars = "▁▂▃▄▅▆▇█"
	// waveformWidth is how many bars a waveform is drawn with at most.
	waveformWidth = 32
	// seekStep is how far a key seeks a voice message.
	seekStep = time.Second
)

// waveformRail returns the bars the waveform of a voice message is drawn with.
func waveformRail(waveform []byte) []string {
	bars := []rune(waveformBars)

	var rail []string
	for chunk := range slices.Chunk(waveform, len(waveform)/waveformWidth+1) {
		rail = append(rail, string(bars[int(slices.Max(chunk))*len(bars)/256]))
	}

	return rail
}

func formatDuration(d time.Duration) string {
	seconds := int(d.Round(time.Second).Seconds())
	return fmt.Sprintf("%d:%02d", seconds/60, seconds%60)
}

// voiceMessageView returns the player of the voice message a: whether it is playing, its waveform with what was played of it, and how long it is.
func (m Model) voiceMessageView(a discord.Attachment) tview.Widget {
	duration := time.Duration(a.DurationSecs * float64(time.Second))
	symbol, label := symbolPlay, formatDuration(duration)

	status := m.voiceStatus
	loaded := status.ID == a.ID
	if loaded {
		label = formatDuration(status.Position) + positionSeparator + label
	}

	if loaded && status.Playing {
		symbol = symbolPause
	}

	rail := waveformRail(a.Waveform)
	style := m.cfg.Theme.MessagesList.AttachmentStyle.Style
	// Only the voice message that is loaded can be sought.
	var onSeek func(time.Duration) tview.Msg
	if loaded {
		onSeek = func(position time.Duration) tview.Msg { return seekMsg(position) }
	}

	waveform := slider.New(0, duration, status.Position, onSeek).
		OnPress(func(position time.Duration) tview.Msg { return seekPressMsg(position) }).
		OnRelease(seekReleaseMsg{}).
		Dragging(loaded && m.seeking).
		Width(layout.Fixed(len(rail))).
		Rail(func(cell, handle int) (string, tview.Style) {
			if loaded && cell <= handle {
				return rail[cell], style
			}

			return rail[cell], style.Dim(true)
		})

	width, _ := text.New(symbol).Size()
	play := button.New().Label(symbol).Width(width).OnClick(playMsg(a))

	return row.New(play, waveform, text.New(label)).Spacing(1).Height(layout.Fixed(1))
}

func (m Model) drawEmbeds(builder *richtext.Builder, message discord.Message, baseStyle tview.Style, width int, contentRoot ast.Node, contentSource []byte) {
	if len(message.Embeds) == 0 {
		return
	}

	// Embed URLs are deduplicated against links already shown in the message body.
	// Reuse the body's parsed AST when markdown is enabled; only the markdown-disabled path (no AST) has to parse the content here.
	var contentListURLs []string
	if contentRoot != nil {
		contentListURLs = urlsFromAST(contentRoot, contentSource)
	} else {
		contentListURLs = extractURLs(message.Content)
	}
	contentURLs := make(map[string]struct{}, len(contentListURLs))
	for _, u := range contentListURLs {
		contentURLs[u] = struct{}{}
	}

	lineStyles := embedLineStyles(baseStyle, m.cfg.Theme.MessagesList.Embeds)
	defaultBarStyle := baseStyle.Dim(true)
	prefixText := "  ▎ "
	prefixWidth := uniseg.StringWidth(prefixText)
	// Wrap against the list's width, so the message is rendered again when it changes.
	wrapWidth := max(width-prefixWidth, 1)

	for _, embed := range message.Embeds {
		lines := embedLines(embed, contentURLs)
		if len(lines) == 0 {
			continue
		}

		embedText := make(richtext.Text, 0, len(lines)*2)
		barStyle := defaultBarStyle
		if embed.Color != discord.NullColor && embed.Color != 0 {
			barStyle = barStyle.Foreground(tcell.NewHexColor(int32(embed.Color)))
		}
		prefix := richtext.NewSegment(prefixText, barStyle)
		builder.NewLine()
		for _, line := range lines {
			if strings.TrimSpace(line.Text) == "" {
				continue
			}
			msg := message
			msg.Content = line.Text
			lineStyle := lineStyles[line.Kind]
			// Embed descriptions are always markdown-rendered to match Discord's rich embed semantics, even when message markdown is globally disabled.
			rendered, _, _ := m.renderContent(msg, lineStyle, line.Kind == embedLineDescription)
			for _, renderedLine := range rendered {
				if line.URL != "" {
					renderedLine = lineWithURL(renderedLine, line.URL)
				}
				// Prefix must be applied after wrapping so every visual line keeps the embed bar marker ("▎"), not only the first logical line.
				for _, wrapped := range richtext.Wrap(renderedLine, wrapWidth) {
					embedText = append(embedText, append(richtext.Line{prefix}, wrapped...))
				}
			}
		}

		if len(embedText) > 0 {
			builder.WriteText(embedText)
		}
	}
}

func lineWithURL(line richtext.Line, rawURL string) richtext.Line {
	out := make(richtext.Line, len(line))
	for i, segment := range line {
		out[i] = segment
		out[i].Style = out[i].Style.Url(rawURL)
	}
	return out
}

type embedLine struct {
	Text string
	Kind embedLineKind
	URL  string
}

type embedLineKind uint8

const (
	// Keep this ordering stable: drawEmbeds indexes precomputed style slots by this enum.
	embedLineProvider embedLineKind = iota
	embedLineAuthor
	embedLineTitle
	embedLineDescription
	embedLineFieldName
	embedLineFieldValue
	embedLineFooter
	embedLineURL
)

func embedLineStyles(baseStyle tview.Style, theme config.MessagesListEmbedsThemeConfig) [8]tview.Style {
	styles := [8]tview.Style{}
	styles[embedLineProvider] = tview.MergeStyle(baseStyle, theme.ProviderStyle.Style)
	styles[embedLineAuthor] = tview.MergeStyle(baseStyle, theme.AuthorStyle.Style)
	styles[embedLineTitle] = tview.MergeStyle(baseStyle, theme.TitleStyle.Style)
	styles[embedLineDescription] = tview.MergeStyle(baseStyle, theme.DescriptionStyle.Style)
	styles[embedLineFieldName] = tview.MergeStyle(baseStyle, theme.FieldNameStyle.Style)
	styles[embedLineFieldValue] = tview.MergeStyle(baseStyle, theme.FieldValueStyle.Style)
	styles[embedLineFooter] = tview.MergeStyle(baseStyle, theme.FooterStyle.Style)
	styles[embedLineURL] = tview.MergeStyle(baseStyle, theme.URLStyle.Style)
	return styles
}

type embedLineDedupKey struct {
	kind embedLineKind
	text string
}

func embedLines(embed discord.Embed, contentURLs map[string]struct{}) []embedLine {
	lines := make([]embedLine, 0, 8)
	seen := make(map[embedLineDedupKey]struct{}, 8)

	appendUnique := func(s string, kind embedLineKind, rawURL string) {
		s = strings.TrimSpace(s)
		if s == "" {
			return
		}
		// Deduplicate by kind+text so the same value can intentionally appear in multiple semantic slots with different styles (e.g. title vs. field).
		key := embedLineDedupKey{kind: kind, text: s}
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		lines = append(lines, embedLine{
			Text: s,
			Kind: kind,
			URL:  rawURL,
		})
	}

	appendURL := func(url discord.URL) {
		u := strings.TrimSpace(url)
		if u == "" {
			return
		}
		// Avoid duplicating links that already appear in message body content.
		if _, ok := contentURLs[u]; ok {
			return
		}
		appendUnique(linkDisplayText(u), embedLineURL, u)
	}

	if embed.Provider != nil {
		appendUnique(embed.Provider.Name, embedLineProvider, "")
	}
	if embed.Author != nil {
		appendUnique(embed.Author.Name, embedLineAuthor, "")
	}
	appendUnique(embed.Title, embedLineTitle, embed.URL)
	// Some Discord embeds include markdown-escaped punctuation in raw payload text (e.g. "\."), so normalize for display.
	appendUnique(unescapeMarkdownEscapes(embed.Description), embedLineDescription, "")

	for _, field := range embed.Fields {
		appendUnique(field.Name, embedLineFieldName, "")
		appendUnique(field.Value, embedLineFieldValue, "")
	}

	if embed.Footer != nil {
		appendUnique(embed.Footer.Text, embedLineFooter, "")
	}

	// Prefer media URLs after textual fields so previews read top-to-bottom before jumping to link targets.
	// When a title exists, embed.URL is represented by title Style.Url metadata instead of a separate URL row.
	if embed.Title == "" {
		appendURL(embed.URL)
	}
	if embed.Image != nil {
		appendURL(embed.Image.URL)
	}
	if embed.Video != nil {
		appendURL(embed.Video.URL)
	}

	return lines
}

func linkDisplayText(raw string) string {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Host == "" {
		return raw
	}

	path := strings.TrimSpace(parsed.EscapedPath())
	switch {
	case path == "", path == "/":
		return parsed.Host
	case len(path) > 48:
		return parsed.Host + path[:45] + "..."
	default:
		return parsed.Host + path
	}
}

func unescapeMarkdownEscapes(s string) string {
	if !strings.ContainsRune(s, '\\') {
		return s
	}

	var b strings.Builder
	b.Grow(len(s))

	for i := range len(s) {
		if s[i] == '\\' && i+1 < len(s) && isMarkdownEscapable(s[i+1]) {
			continue
		}
		b.WriteByte(s[i])
	}
	return b.String()
}

func isMarkdownEscapable(c byte) bool {
	switch c {
	case '\\', '`', '*', '_', '{', '}', '[', ']', '(', ')', '#', '+', '-', '.', '!', '|', '>', '~':
		return true
	default:
		return false
	}
}

func (m Model) drawForwardedMessage(builder *richtext.Builder, message discord.Message, baseStyle tview.Style, grouped bool) {
	dimStyle := baseStyle.Dim(true)
	m.drawLayout(builder, message, baseStyle, grouped, func() {
		builder.Write(m.cfg.UI.MessagesList.ForwardedIndicator+" ", dimStyle)
		m.drawSnapshotContent(builder, message, message.MessageSnapshots[0].Message, baseStyle)
		builder.Write(" ("+m.formatTimestamp(message.MessageSnapshots[0].Message.Timestamp)+") ", dimStyle)
	})
}

func (m Model) drawReply(builder *richtext.Builder, message discord.Message, baseStyle tview.Style) {
	dimStyle := baseStyle.Dim(true)
	builder.Write(m.cfg.UI.MessagesList.ReplyIndicator+" ", dimStyle)

	if reference := message.ReferencedMessage; reference != nil {
		reference.GuildID = message.GuildID
		m.drawAuthor(builder, *reference, dimStyle)
		builder.Write(" ", dimStyle)
		m.drawContent(builder, *reference, dimStyle)
	} else {
		builder.Write("Original message was deleted", dimStyle)
	}
}

func (m Model) drawPinnedMessage(builder *richtext.Builder, message discord.Message, baseStyle tview.Style) {
	builder.Write(message.Author.DisplayOrUsername()+" pinned a message.", baseStyle)
}

func (m Model) selectedMessage() (*discord.Message, bool) {
	cursor := m.cursor
	if cursor < 0 || cursor >= len(m.items) || m.items[cursor].separator {
		return nil, false
	}

	return &m.items[cursor].message, true
}

// Update returns ml changed in response to msg and a command to run, or nil.
func (m Model) Update(msg tview.Msg) (Model, tview.Cmd) {
	switch msg := msg.(type) {
	case tview.KeyMsg:
		switch {
		case keybind.Matches(msg, m.cfg.Keybinds.MessagesList.Cancel.Keybind):
			m.clearSelection()
		case keybind.Matches(msg, m.cfg.Keybinds.MessagesList.SelectUp.Keybind):
			return m, m.selectUp()
		case keybind.Matches(msg, m.cfg.Keybinds.MessagesList.SelectDown.Keybind):
			m.selectDown()
		case keybind.Matches(msg, m.cfg.Keybinds.MessagesList.SelectTop.Keybind):
			return m, m.selectTop()
		case keybind.Matches(msg, m.cfg.Keybinds.MessagesList.SelectBottom.Keybind):
			m.selectBottom()
		case keybind.Matches(msg, m.cfg.Keybinds.MessagesList.SelectReply.Keybind):
			m.selectReply()
		case keybind.Matches(msg, m.cfg.Keybinds.MessagesList.YankID.Keybind):
			return m, m.yankMessageID()
		case keybind.Matches(msg, m.cfg.Keybinds.MessagesList.YankContent.Keybind):
			return m, m.yankContent()
		case keybind.Matches(msg, m.cfg.Keybinds.MessagesList.YankURL.Keybind):
			return m, m.yankURL()
		case keybind.Matches(msg, m.cfg.Keybinds.MessagesList.Open.Keybind):
			return m, m.open()
		case keybind.Matches(msg, m.cfg.Keybinds.MessagesList.OpenInBrowser.Keybind):
			return m, m.openInBrowser()
		case keybind.Matches(msg, m.cfg.Keybinds.MessagesList.Play.Keybind):
			if message, ok := m.selectedMessage(); ok && isVoiceMessage(*message) {
				return m, m.play(message.Attachments[0])
			}
		case keybind.Matches(msg, m.cfg.Keybinds.MessagesList.SeekBackward.Keybind):
			return m, m.seek(m.voiceStatus.Position - seekStep)
		case keybind.Matches(msg, m.cfg.Keybinds.MessagesList.SeekForward.Keybind):
			return m, m.seek(m.voiceStatus.Position + seekStep)
		case keybind.Matches(msg, m.cfg.Keybinds.MessagesList.Download.Keybind):
			return m, m.download()
		case keybind.Matches(msg, m.cfg.Keybinds.MessagesList.Reply.Keybind):
			return m, m.reply(false)
		case keybind.Matches(msg, m.cfg.Keybinds.MessagesList.ReplyMention.Keybind):
			return m, m.reply(true)
		case keybind.Matches(msg, m.cfg.Keybinds.MessagesList.Edit.Keybind):
			return m, m.editSelectedMessage()
		case keybind.Matches(msg, m.cfg.Keybinds.MessagesList.Delete.Keybind):
			return m, m.deleteSelectedMessage()
		case keybind.Matches(msg, m.cfg.Keybinds.MessagesList.DeleteConfirm.Keybind):
			return m, m.confirmDelete()
		}
	case olderMessagesLoadedMsg:
		selectedChannel := m.channel
		if selectedChannel == nil || selectedChannel.ID != msg.ChannelID {
			return m, nil
		}
		older := make([]messageItem, len(msg.Older))
		for i, message := range msg.Older {
			older[i] = messageItem{message: message}
		}
		m.items = slices.Concat(older, m.items)
		// The selected and target messages stay the same, so they are not scrolled to.
		if m.cursor >= 0 {
			m.cursor += len(older)
		}
		if m.target >= 0 {
			m.target += len(older)
		}
		m.rebuildItems()
		if selectedChannel.GuildID.IsValid() {
			return m, tview.Batch(m.requestGuildMembers(selectedChannel.GuildID, msg.Older), m.loadPreviews())
		}
		return m, m.loadPreviews()
	case previewLoadedMsg:
		preview, cmd := msg.preview, tview.Cmd(nil)
		if m.kitty {
			m.kittyImages++
			preview = preview.Kitty(m.kittyImages%63488 + 1)
			cmd = preview.Transmit()
		}

		if _, ok := m.avatars[msg.proxy]; ok {
			m.avatars[msg.proxy] = preview
			return m, cmd
		}

		index := slices.IndexFunc(m.items, func(item messageItem) bool { return !item.separator && item.message.ID == msg.messageID })
		if index < 0 || m.items[index].previews == nil {
			return m, nil
		}

		m.items[index].previews[msg.proxy] = preview
		return m, cmd
	case playMsg:
		return m, m.play(discord.Attachment(msg))
	case seekPressMsg:
		m.seeking = true
		m.voiceStatus.Position = time.Duration(msg)
		return m, nil
	case seekMsg:
		m.voiceStatus.Position = time.Duration(msg)
		if m.seeking {
			return m, nil
		}

		return m, m.seek(m.voiceStatus.Position)
	case seekReleaseMsg:
		m.seeking = false
		return m, m.seek(m.voiceStatus.Position)
	case voiceMsg:
		status := m.voicePlayer.Status()
		// The waveform shows where it is dragged to rather than what is playing, while that voice message is the one loaded.
		m.seeking = m.seeking && status.ID == m.voiceStatus.ID
		if m.seeking {
			status.Position = m.voiceStatus.Position
		}

		m.voiceStatus = status
		return m, m.Init()
	case tview.ResizeMsg:
		cols, rows := msg.Size()
		width, height := msg.PixelSize()
		if cols > 0 && rows > 0 {
			m.cellWidth, m.cellHeight = uint(width/cols), uint(height/rows)
		}
	case tview.TerminalInfoMsg:
		switch m.cfg.UI.MessagesList.ImageProtocol {
		case "kitty":
			m.kitty = true
		case "halfblocks":
			m.kitty = false
		default:
			// WezTerm and Konsole support kitty graphics but not Unicode placeholders.
			m.kitty = msg.Name == "kitty" || msg.Name == "ghostty"
		}
	case scrollMsg:
		m.scrollState = viewport.ScrollState(msg)
		// Older messages are fetched once as the list is scrolled to the top, rather than on each scroll there.
		if !scrolledToTop(m.scrollState.Origin()) && scrolledToTop(m.scrollState.Viewport()) {
			return m, m.fetchOlderMessages()
		}
	case listMsg:
		top := m.atTop()
		m.cursor, m.target = int(msg), int(msg)
		m.onRowCursorChanged(m.cursor)
		if !top && m.atTop() {
			return m, m.fetchOlderMessages()
		}
	}
	return m, nil
}

// scrolledToTop reports whether v is scrolled to the oldest message shown, which is false while the messages fit, as there is nothing to scroll.
func scrolledToTop(v viewport.Viewport) bool {
	return v.Content().Height > v.Bounds().Height && v.AbsoluteOffsetReversed().Y == 0
}

// atTop reports whether the oldest message shown is selected.
func (m Model) atTop() bool {
	return m.cursor >= 0 && m.cursor == m.messageIndex(-1, 1)
}

func (m *Model) selectDown() {
	if m.cursor == -1 {
		m.selectBottom()
	} else if next := m.messageIndex(m.cursor, 1); next >= 0 {
		m.setCursor(next)
	}
}

func (m *Model) selectTop() tview.Cmd {
	if m.atTop() {
		return nil
	}
	m.setCursor(m.messageIndex(-1, 1))
	return m.fetchOlderMessages()
}

func (m *Model) selectBottom() {
	m.setCursor(m.messageIndex(len(m.items), -1))
}

func (m *Model) selectReply() {
	messages := m.items
	cursor := m.cursor
	if cursor < 0 || cursor >= len(messages) {
		return
	}

	if ref := messages[cursor].message.ReferencedMessage; ref != nil {
		refIdx := slices.IndexFunc(messages, func(m messageItem) bool {
			return !m.separator && m.message.ID == ref.ID
		})
		if refIdx != -1 {
			m.setCursor(refIdx)
		}
	}
}

func extractURLs(content string) []string {
	src := []byte(content)
	node := md.Parse(src)
	return urlsFromAST(node, src)
}

// urlsFromAST collects link destinations from an already-parsed markdown AST.
// src must be the byte slice the node was parsed from (AutoLink resolves its URL against it).
func urlsFromAST(node ast.Node, src []byte) []string {
	var urls []string
	_ = ast.Walk(node, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if entering {
			switch n := n.(type) {
			case *ast.AutoLink:
				urls = append(urls, string(n.Destination.Bytes(src)))
			case *ast.Link:
				urls = append(urls, string(n.Destination.Bytes(src)))
			}
		}

		return ast.WalkContinue, nil
	})
	return urls
}

func extractEmbedURLs(embeds []discord.Embed) []string {
	urls := make([]string, 0, len(embeds)*3)
	for _, embed := range embeds {
		if embed.URL != "" {
			urls = append(urls, embed.URL)
		}
		if embed.Image != nil && embed.Image.URL != "" {
			urls = append(urls, embed.Image.URL)
		}
		if embed.Video != nil && embed.Video.URL != "" {
			urls = append(urls, embed.Video.URL)
		}
	}
	return urls
}

// The schemes of the links that are opened.
const (
	schemeHTTP  = "http"
	schemeHTTPS = "https"
)

func messageURLs(msg discord.Message) []string {
	combined := slices.Concat(extractURLs(msg.Content), extractEmbedURLs(msg.Embeds))

	urls := make([]string, 0, len(combined))
	seen := make(map[string]struct{}, len(combined))
	for _, u := range combined {
		u = strings.TrimSpace(u)
		// Only web links are opened, as one of another scheme could run a program.
		if parsed, err := url.Parse(u); err != nil || parsed.Scheme != schemeHTTP && parsed.Scheme != schemeHTTPS {
			continue
		}
		if _, ok := seen[u]; ok {
			continue
		}
		seen[u] = struct{}{}
		urls = append(urls, u)
	}
	return urls
}

// get requests the file at url, by a new link to it if that one expired.
func (m Model) get(url string) (*http.Response, error) {
	resp, err := http.Get(url)
	if err != nil || resp.StatusCode == http.StatusOK {
		return resp, err
	}
	resp.Body.Close() //nolint:errcheck

	var links struct {
		URLs []struct {
			Refreshed string `json:"refreshed"`
		} `json:"refreshed_urls"`
	}
	body := map[string][]string{"attachment_urls": {url}}
	if err := m.state.RequestJSONInto(&links, http.MethodPost, api.Endpoint+"attachments/refresh-urls", httputil.WithJSONBody(body)); err != nil {
		return nil, err
	}

	if len(links.URLs) != 0 {
		if resp, err = http.Get(links.URLs[0].Refreshed); err != nil || resp.StatusCode == http.StatusOK {
			return resp, err
		}
		resp.Body.Close() //nolint:errcheck
	}

	return nil, fmt.Errorf("unexpected HTTP status: %s", resp.Status)
}

func (m Model) downloadAttachment(attachment discord.Attachment, destination string) error {
	resp, err := m.get(attachment.URL)
	if err != nil {
		return err
	}
	defer resp.Body.Close() //nolint:errcheck
	file, err := os.Create(destination)
	if err != nil {
		return err
	}
	_, err = io.Copy(file, resp.Body)
	return errors.Join(err, file.Close())
}

// SelectLastOwn selects the last message in the selected channel that the user can edit, and returns it.
func (m *Model) SelectLastOwn() (discord.Message, bool) {
	channel := m.channel
	if channel == nil {
		return discord.Message{}, false
	}
	for i, item := range slices.Backward(m.items) {
		message := item.message
		if item.separator || message.ChannelID != channel.ID || !ui.IsMe(m.state, message.Author.ID) ||
			!message.ID.IsValid() || message.Content == "" || len(message.MessageSnapshots) > 0 ||
			(message.Type != discord.DefaultMessage && message.Type != discord.InlinedReplyMessage) {
			continue
		}
		m.setCursor(i)
		return message, true
	}
	return discord.Message{}, false
}

func (m Model) canDeleteMessage(message discord.Message) bool {
	return ui.IsMe(m.state, message.Author.ID) ||
		(message.GuildID.IsValid() && m.state.HasPermissions(message.ChannelID, discord.PermissionManageMessages))
}

func (m Model) InvalidateRendered() {
	for i := range m.items {
		m.items[i].text = nil
	}
}
