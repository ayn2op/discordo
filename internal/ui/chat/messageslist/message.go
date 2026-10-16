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
	"github.com/ayn2op/tview"
	"github.com/ayn2op/tview/layout"
	"github.com/ayn2op/tview/richtext"
	"github.com/ayn2op/tview/textview"
	"github.com/gdamore/tcell/v3"
	"github.com/gdamore/tcell/v3/color"
	"github.com/yuin/goldmark/v2/ast"
)

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
		drawPinnedMessage(builder, message, baseStyle)
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

func drawPinnedMessage(builder *richtext.Builder, message discord.Message, baseStyle tview.Style) {
	builder.Write(message.Author.DisplayOrUsername()+" pinned a message.", baseStyle)
}
