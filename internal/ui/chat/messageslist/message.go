package messageslist

import (
	"log/slog"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/ayn2op/arikawa/v3/discord"
	md "github.com/ayn2op/arikawa/v3/markdown"
	"github.com/ayn2op/arikawa/v3/state"
	"github.com/ayn2op/discordo/internal/markdown"
	"github.com/ayn2op/tview"
	"github.com/ayn2op/tview/layout"
	"github.com/ayn2op/tview/row"
	"github.com/ayn2op/tview/text"
	"github.com/gdamore/tcell/v3"
	"github.com/gdamore/tcell/v3/color"
	"github.com/yuin/goldmark/v2/ast"
)

// messageViews returns the views of the text of message and of what it replies to.
// They capture the model and the message, so they are made here: capturing them in buildItem would copy both to the heap on every call, as the list builds every item each time it is laid out.
// root is the parsed content of message, or nil to parse it when it is rendered.
func (m Model) messageViews(message discord.Message, root ast.Node, grouped bool) (content, reply *messageView) {
	style := m.cfg.Theme.MessagesList.MessageStyle.Style
	content = &messageView{render: func(width int) []text.Span { return m.renderMessage(message, root, style, width, grouped) }}
	reply = &messageView{render: func(int) []text.Span {
		builder := new(markdown.Builder)
		m.drawReply(builder, message, style)
		return builder.Finish()
	}}
	return content, reply
}

func (m Model) blocked(user discord.User) bool {
	return m.cfg.HideBlockedUsers && m.state.UserIsBlocked(user.ID)
}

// messageView is the text of a message. It keeps the wrapped lines until the width changes, as the list measures every message when it draws.
type messageView struct {
	render   func(width int) []text.Span
	width    int
	rendered bool
	lines    [][]text.Span
}

func (v *messageView) wrap(width int) {
	if !v.rendered || v.width != width {
		v.lines, v.width, v.rendered = text.Lines(v.render(width), width, text.WrappingWordOrGlyph), width, true
	}
}

func (v *messageView) Size() (width, height layout.Length) { return layout.Fill, layout.Shrink }

func (v *messageView) Layout(limits layout.Limits) layout.Size {
	return layout.Sized(limits, layout.Fill, layout.Shrink, func(limits layout.Limits) layout.Size {
		v.wrap(limits.Max.Width)
		return layout.Size{Height: len(v.lines)}
	})
}

func (v *messageView) Draw(screen tview.Screen, area tview.Rectangle) {
	v.wrap(area.Width)
	for i, line := range v.lines[:min(len(v.lines), area.Height)] {
		text.Rich(line...).Wrapping(text.WrappingNone).Draw(screen, tview.Rectangle{X: area.X, Y: area.Y + i, Width: area.Width, Height: 1})
	}
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
	tview.Print(screen, line, area.X, area.Y, area.Width, tview.HorizontalAlignmentLeft, d.style)
}

func (dateSeparator) Handle(msg tview.Msg, area tview.Rectangle) tview.Msg { return msg }

func (m Model) renderMessage(message discord.Message, root ast.Node, baseStyle tview.Style, width int, grouped bool) []text.Span {
	builder := new(markdown.Builder)
	m.writeMessage(builder, message, root, baseStyle, width, grouped)
	// Without a header or content, the message starts with the line break that attachments and embeds start below.
	return trimLeadingLineBreaks(builder.Finish())
}

// trimLeadingLineBreaks returns spans without the line breaks they start with.
func trimLeadingLineBreaks(spans []text.Span) []text.Span {
	for len(spans) > 0 {
		if spans[0].Text = strings.TrimLeft(spans[0].Text, "\n"); spans[0].Text != "" {
			break
		}
		spans = spans[1:]
	}
	return spans
}

func (m Model) writeMessage(builder *markdown.Builder, message discord.Message, root ast.Node, baseStyle tview.Style, width int, grouped bool) {
	if m.blocked(message.Author) {
		builder.Write("Blocked message", baseStyle.Foreground(color.Red).Bold(true))
		return
	}

	switch message.Type {
	case discord.DefaultMessage:
		if message.Reference != nil && message.Reference.Type == discord.MessageReferenceTypeForward {
			m.drawForwardedMessage(builder, message, baseStyle, grouped)
		} else {
			m.drawDefaultMessage(builder, message, root, baseStyle, width, grouped)
		}
	case discord.GuildMemberJoinMessage:
		m.drawLayout(builder, message, baseStyle, false, func() { builder.Write("joined the server.", baseStyle) })
	case discord.InlinedReplyMessage:
		m.drawDefaultMessage(builder, message, root, baseStyle, width, false)
	case discord.ChannelPinnedMessage:
		drawPinnedMessage(builder, message, baseStyle)
	default:
		m.drawLayout(builder, message, baseStyle, false, nil)
	}
}

// reactionsView returns the reactions wrapped to the width of the message, with the image of each custom emoji that has one before its name, or nil if there are none.
func (m Model) reactionsView(reactions []discord.Reaction) tview.Widget {
	if len(reactions) == 0 {
		return nil
	}

	theme, cfg := m.cfg.Theme.MessagesList, m.cfg.UI.MessagesList.Reactions
	separatorStyle := tview.MergeStyle(theme.MessageStyle.Style, theme.ReactionStyle.Style)
	chips := make([]tview.Widget, len(reactions))
	for i, reaction := range reactions {
		style := theme.ReactionStyle.Style
		if reaction.Me {
			style = theme.OwnReactionStyle.Style
		}
		style = tview.MergeStyle(theme.MessageStyle.Style, style)

		label := strconv.Itoa(reaction.Count)
		emoji, hasImage := m.images[emojiURL(reaction.Emoji.ID)]
		if !hasImage || cfg.ShowEmojiName {
			name := reaction.Emoji.Name
			if reaction.Emoji.IsCustom() {
				name = cfg.Name(name)
			}
			label = name + " " + label
		}

		spans := []text.Span{{Text: label, Style: style}}
		// Each reaction but the last ends with the separator, so a line never starts with one.
		if i < len(reactions)-1 {
			spans = append(spans, text.Span{Text: cfg.Separator, Style: separatorStyle})
		}

		var chip tview.Widget = text.Rich(spans...)
		if hasImage {
			chip = row.New(emoji, chip).Spacing(1).Width(layout.Shrink).Height(layout.Shrink)
		}
		chips[i] = chip
	}
	return row.New(chips...).Height(layout.Shrink).Wrap()
}

func (m Model) formatTimestamp(ts discord.Timestamp) string {
	return ts.Time().In(time.Local).Format(m.cfg.Timestamps.Format)
}

const editedIndicator = " (edited)"

// drawLayout draws the header of message, unless it is grouped, and its content.
func (m Model) drawLayout(builder *markdown.Builder, message discord.Message, baseStyle tview.Style, grouped bool, content func()) {
	if m.cfg.UI.MessagesList.Compact {
		m.drawCompact(builder, message, baseStyle, grouped, content)
		return
	}

	m.drawCozy(builder, message, baseStyle, grouped, content)
}

func (m Model) drawCompact(builder *markdown.Builder, message discord.Message, baseStyle tview.Style, grouped bool, content func()) {
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

func (m Model) drawCozy(builder *markdown.Builder, message discord.Message, baseStyle tview.Style, grouped bool, content func()) {
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

func (m Model) drawAuthor(builder *markdown.Builder, message discord.Message, baseStyle tview.Style) {
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

// drawContent renders the message body. root is its parsed content, or nil to parse it here.
func (m Model) drawContent(builder *markdown.Builder, message discord.Message, root ast.Node, baseStyle tview.Style) {
	if root == nil && m.cfg.Markdown.Enabled {
		root = m.parse(message)
	}

	content := m.renderContent(message, root, baseStyle, false)
	if m.cfg.Markdown.Enabled {
		startsWithCodeBlock := false
		if first := root.FirstChild(); first != nil {
			_, startsWithCodeBlock = first.(*ast.CodeBlock)
		}

		// Keep code blocks visually separate from "timestamp + author".
		if startsWithCodeBlock && !builder.LineEmpty() {
			builder.NewLine()
		}

		content = trimLeadingLineBreaks(content)
	}
	builder.WriteSpans(content...)
}

// root is the parsed content of message, or nil to parse it here.
func (m Model) renderContent(message discord.Message, root ast.Node, baseStyle tview.Style, forceMarkdown bool) []text.Span {
	// Keep one rendering path for both normal messages and embed fragments so we preserve mention/link parsing behavior consistently across both.
	if forceMarkdown || m.cfg.Markdown.Enabled {
		if root == nil {
			root = m.parse(message)
		}

		return m.renderer.RenderText([]byte(message.Content), root, baseStyle, m.emojiLine)
	}

	b := new(markdown.Builder)
	b.Write(message.Content, baseStyle)
	return b.Finish()
}

func (m Model) drawSnapshotContent(builder *markdown.Builder, parent discord.Message, snapshot discord.MessageSnapshotMessage, baseStyle tview.Style) {
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
	m.drawContent(builder, message, nil, baseStyle)
}

func (m Model) drawDefaultMessage(builder *markdown.Builder, message discord.Message, root ast.Node, baseStyle tview.Style, width int, grouped bool) {
	content := func() { m.drawContent(builder, message, root, baseStyle) }
	if message.Content == "" {
		content = nil
	}

	m.drawLayout(builder, message, baseStyle, grouped, content)
	m.drawEmbeds(builder, message, baseStyle, width, root)

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

func (m Model) drawForwardedMessage(builder *markdown.Builder, message discord.Message, baseStyle tview.Style, grouped bool) {
	dimStyle := baseStyle.Dim(true)
	m.drawLayout(builder, message, baseStyle, grouped, func() {
		builder.Write(m.cfg.UI.MessagesList.ForwardedIndicator+" ", dimStyle)
		m.drawSnapshotContent(builder, message, message.MessageSnapshots[0].Message, baseStyle)
		builder.Write(" ("+m.formatTimestamp(message.MessageSnapshots[0].Message.Timestamp)+") ", dimStyle)
	})
}

func (m Model) drawReply(builder *markdown.Builder, message discord.Message, baseStyle tview.Style) {
	dimStyle := baseStyle.Dim(true)
	builder.Write(m.cfg.UI.MessagesList.ReplyIndicator+" ", dimStyle)

	if reference := message.ReferencedMessage; reference != nil {
		reference.GuildID = message.GuildID
		m.drawAuthor(builder, *reference, dimStyle)
		builder.Write(" ", dimStyle)
		m.drawContent(builder, *reference, nil, dimStyle)
	} else {
		builder.Write("Original message was deleted", dimStyle)
	}
}

func drawPinnedMessage(builder *markdown.Builder, message discord.Message, baseStyle tview.Style) {
	builder.Write(message.Author.DisplayOrUsername()+" pinned a message.", baseStyle)
}

// parse returns the parsed markdown of the content of message, with its mentions resolved from the state.
func (m Model) parse(message discord.Message) ast.Node {
	return md.ParseWithMessage([]byte(message.Content), *m.state.Cabinet, &message)
}
