package messageslist

import (
	"log/slog"
	"strconv"
	"strings"
	"time"

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

func (m Model) blocked(user discord.User) bool {
	return m.cfg.HideBlockedUsers && m.state.UserIsBlocked(user.ID)
}

// messageText returns spans as text that fills its width, keeping its wrapped lines until the width changes, as the list measures every message when it draws.
func messageText(spans []text.Span) tview.Widget {
	return text.Rich(spans...).Width(layout.Fill).Wrapping(text.WrappingWordOrGlyph).Paragraph(new(text.Paragraph))
}

func (m Model) renderMessage(message discord.Message, root ast.Node, baseStyle tview.Style, grouped bool) []text.Span {
	builder := new(markdown.Builder)
	m.writeMessage(builder, message, root, baseStyle, grouped)
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

func (m Model) writeMessage(builder *markdown.Builder, message discord.Message, root ast.Node, baseStyle tview.Style, grouped bool) {
	if m.blocked(message.Author) {
		builder.Write("Blocked message", baseStyle.Foreground(color.Red).Bold(true))
		return
	}

	switch message.Type {
	case discord.DefaultMessage:
		if message.Reference != nil && message.Reference.Type == discord.MessageReferenceTypeForward {
			m.drawForwardedMessage(builder, message, baseStyle, grouped)
		} else {
			m.drawDefaultMessage(builder, message, root, baseStyle, grouped)
		}
	case discord.GuildMemberJoinMessage:
		m.drawLayout(builder, message, baseStyle, false, func() { builder.Write("joined the server.", baseStyle) })
	case discord.InlinedReplyMessage:
		m.drawDefaultMessage(builder, message, root, baseStyle, false)
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
	for index, reaction := range reactions {
		style := theme.ReactionStyle.Style
		if reaction.Me {
			style = theme.OwnReactionStyle.Style
		}

		style = tview.MergeStyle(theme.MessageStyle.Style, style)

		emoji, hasImage := m.images[emojiURL(reaction.Emoji.ID)]
		spans := []text.Span{{Text: m.reactionLabel(reaction, hasImage), Style: style}}
		// Each reaction but the last ends with the separator, so a line never starts with one.
		if index < len(reactions)-1 {
			spans = append(spans, text.Span{Text: cfg.Separator, Style: separatorStyle})
		}

		var chip tview.Widget = text.Rich(spans...)
		if hasImage {
			chip = row.New(emoji, chip).Spacing(1).Width(layout.Shrink).Height(layout.Shrink)
		}
		chips[index] = chip
	}

	return row.New(chips...).Height(layout.Shrink).Wrap()
}

// reactionLabel returns the count of reaction, after the name of its emoji unless its image is shown alone.
func (m Model) reactionLabel(reaction discord.Reaction, hasImage bool) string {
	cfg := m.cfg.UI.MessagesList.Reactions
	label := strconv.Itoa(reaction.Count)
	if hasImage && !cfg.ShowEmojiName {
		return label
	}

	name := reaction.Emoji.Name
	if reaction.Emoji.IsCustom() {
		name = cfg.Name(name)
	}

	return name + " " + label
}

func (m Model) formatTimestamp(timestamp discord.Timestamp) string {
	return timestamp.Time().In(time.Local).Format(m.cfg.Timestamps.Format)
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
			role, _ := m.state.Cabinet.Role(message.GuildID, id)
			return role
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

	builder := new(markdown.Builder)
	builder.Write(message.Content, baseStyle)
	return builder.Finish()
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

func (m Model) drawDefaultMessage(builder *markdown.Builder, message discord.Message, root ast.Node, baseStyle tview.Style, grouped bool) {
	content := func() { m.drawContent(builder, message, root, baseStyle) }
	if message.Content == "" {
		content = nil
	}

	m.drawLayout(builder, message, baseStyle, grouped, content)

	attachmentStyle := tview.MergeStyle(baseStyle, m.cfg.Theme.MessagesList.AttachmentStyle.Style)
	// A voice message is shown as its player.
	if isVoiceMessage(message) {
		return
	}

	for _, attachment := range message.Attachments {
		builder.NewLine()
		if !m.cfg.UI.MessagesList.Attachments.ShowLinks {
			builder.Write(attachment.Filename, attachmentStyle)
			continue
		}

		builder.Write(attachment.Filename+":", attachmentStyle)
		builder.NewLine()
		builder.Write(attachment.URL, attachmentStyle.Url(attachment.URL))
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

	if ref := message.ReferencedMessage; ref != nil {
		ref.GuildID = message.GuildID
		m.drawAuthor(builder, *ref, dimStyle)
		builder.Write(" ", dimStyle)
		m.drawContent(builder, *ref, nil, dimStyle)
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
