package messageslist

import (
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

	"github.com/ayn2op/arikawa/v3/discord"
	md "github.com/ayn2op/arikawa/v3/markdown"
	"github.com/ayn2op/arikawa/v3/state"
	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/discordo/internal/markdown"
	"github.com/ayn2op/discordo/internal/ui"
	"github.com/ayn2op/ningen/v3"
	"github.com/ayn2op/tview"
	"github.com/ayn2op/tview/help"
	tviewimage "github.com/ayn2op/tview/image"
	"github.com/ayn2op/tview/keybind"
	"github.com/ayn2op/tview/list"
	"github.com/ayn2op/tview/richtext"
	"github.com/ayn2op/tview/scrollbar"
	"github.com/ayn2op/tview/textview"
	"github.com/gdamore/tcell/v3"
	"github.com/gdamore/tcell/v3/color"
	"github.com/rivo/uniseg"
	"github.com/yuin/goldmark/ast"
)

type messageItem struct {
	message   discord.Message
	view      list.Item
	separator bool
	timestamp discord.Timestamp

	// A preview is blank while loading and empty if loading failed.
	previews map[discord.URL]tviewimage.Widget
}

type Model struct {
	selectionState list.SelectionState

	scrollBar           scrollbar.Widget
	scrollBarVisibility list.ScrollBarVisibility

	cfg   *config.Config
	state *ningen.State
	// channel is the selected channel, or nil for none.
	channel *discord.Channel
	items   []messageItem

	renderer *markdown.Renderer

	// renderWidth is the width the message being rendered is laid out at.
	renderWidth int

	// kitty reports whether previews are drawn with kitty's graphics protocol.
	// kittyImages counts the images sent to the terminal.
	kitty       bool
	kittyImages int
}

var _ help.KeyMap = Model{}

func NewModel(cfg *config.Config, state *ningen.State) Model {
	ml := Model{
		selectionState: list.NewSelectionState(),
		cfg:            cfg,
		state:          state,
		renderer:       markdown.NewRenderer(cfg),
	}

	ml.selectionState.SetTrackEnd(true)
	ml.selectionState.ScrollToEnd()
	ml.scrollBarVisibility = cfg.Theme.ScrollBar.Visibility.ScrollBarVisibility
	ml.scrollBar = scrollbar.New().
		TrackStyle(cfg.Theme.ScrollBar.TrackStyle.Style).
		ThumbStyle(cfg.Theme.ScrollBar.ThumbStyle.Style).
		GlyphSet(cfg.Theme.ScrollBar.GlyphSet.GlyphSet).
		Arrows(scrollbar.ArrowsBoth)
	return ml
}

// View shows the messages in a box titled with the channel and footed with footer, such as who is typing.
func (ml Model) View(focused bool, footer string) tview.Element {
	return ui.Box(ml.listView(focused), &ml.cfg.Theme, focused).Title(ml.title()).Footer(footer)
}

// title returns the selected channel and its topic, or "Messages" when none is selected.
func (ml *Model) title() string {
	if ml.channel == nil {
		return "Messages"
	}
	title := ui.ChannelToString(*ml.channel, ml.cfg.Icons, ml.state)
	if topic := ml.channel.Topic; topic != "" {
		title += " - " + topic
	}
	return title
}

func (ml *Model) listView(focused bool) list.Widget {
	return list.New(&ml.selectionState, len(ml.items), ml.buildItem).
		SelectedStyle(ml.cfg.Theme.MessagesList.SelectedMessageStyle.Style).
		ScrollBar(ml.scrollBar, ml.scrollBarVisibility).
		Keybind(scrollKeybind(ml.cfg.Keybinds.MessagesList.ScrollKeybinds)).
		Focused(focused).
		OnChange(func(a list.Change) tview.Msg { return listMsg(a) })
}

// scrollKeybind binds kbs to scrolling the list.
func scrollKeybind(kbs config.ScrollKeybinds) func(tview.KeyMsg) (list.Action, bool) {
	return func(key tview.KeyMsg) (list.Action, bool) {
		switch {
		case keybind.Matches(key, kbs.ScrollUp.Keybind):
			return list.ActionScrollUp, true
		case keybind.Matches(key, kbs.ScrollDown.Keybind):
			return list.ActionScrollDown, true
		case keybind.Matches(key, kbs.ScrollTop.Keybind):
			return list.ActionScrollTop, true
		case keybind.Matches(key, kbs.ScrollBottom.Keybind):
			return list.ActionScrollBottom, true
		}
		return 0, false
	}
}

func (ml *Model) cursor() int {
	return ml.selectionState.Cursor()
}

func (ml *Model) setCursor(index int) {
	ml.selectionState.SetCursor(index)
}

// SetChannel shows messages of channel, newest first as Discord sends them, scrolled to the newest, and requests the members who wrote them.
func (ml *Model) SetChannel(channel *discord.Channel, messages []discord.Message) tview.Cmd {
	ml.channel = channel
	ml.reset()
	ml.setMessages(messages)
	if channel.GuildID.IsValid() {
		return tview.Batch(ml.requestGuildMembers(channel.GuildID, messages), ml.loadPreviews())
	}
	return ml.loadPreviews()
}

// ShowNewest clears the selection and scrolls to the newest message, which then stays in view as messages arrive until the user scrolls up.
func (ml *Model) ShowNewest() {
	ml.clearSelection()
	ml.selectionState.ScrollToEnd()
}

// UpdateMessage replaces the shown message with the same ID.
func (ml *Model) UpdateMessage(message discord.Message) tview.Cmd {
	ml.setMessage(ml.indexOf(message.ChannelID, message.ID), message)
	return ml.loadPreviews()
}

// RefreshMessage shows the cached state of a message, such as after a reaction.
func (ml *Model) RefreshMessage(channelID discord.ChannelID, id discord.MessageID) {
	if message, err := ml.state.Cabinet.Message(channelID, id); err == nil {
		ml.setMessage(ml.indexOf(channelID, id), *message)
	}
}

// DeleteMessage removes the shown message with id.
func (ml *Model) DeleteMessage(channelID discord.ChannelID, id discord.MessageID) {
	ml.deleteMessage(ml.indexOf(channelID, id))
}

// indexOf returns the index of the message with id in the channel, or -1 if it is not shown.
func (ml *Model) indexOf(channelID discord.ChannelID, id discord.MessageID) int {
	if ml.channel == nil || ml.channel.ID != channelID {
		return -1
	}
	return slices.IndexFunc(ml.items, func(item messageItem) bool { return !item.separator && item.message.ID == id })
}

func (ml *Model) reset() {
	ml.items = nil
	ml.selectionState = list.NewSelectionState()
	ml.selectionState.SetTrackEnd(true)
	ml.selectionState.ScrollToEnd()
}

func (ml *Model) setMessages(messages []discord.Message) {
	ml.items = make([]messageItem, 0, len(messages))
	for _, message := range slices.Backward(messages) {
		ml.items = append(ml.items, messageItem{message: message})
	}
	ml.rebuildItems()
}

func (ml *Model) AddMessage(message discord.Message) tview.Cmd {
	ml.items = append(ml.items, messageItem{message: message})
	ml.rebuildItems()
	return ml.loadPreviews()
}

func (ml *Model) setMessage(index int, message discord.Message) {
	if index < 0 || index >= len(ml.items) || ml.items[index].separator {
		return
	}

	ml.items[index] = messageItem{message: message, previews: ml.items[index].previews}
	ml.rebuildItems()
}

func (ml *Model) deleteMessage(index int) {
	if index < 0 || index >= len(ml.items) || ml.items[index].separator {
		return
	}

	cursor := ml.cursor()
	if cursor == index {
		cursor = ml.messageIndex(index, -1)
		if cursor == -1 {
			cursor = ml.messageIndex(index, 1)
		}
	}
	if cursor > index {
		cursor--
	}
	ml.items = slices.Delete(ml.items, index, index+1)
	ml.setCursor(cursor)
	ml.rebuildItems()
}

func (ml *Model) clearSelection() {
	ml.setCursor(-1)
}

func (ml *Model) buildItem(index int) list.Item {
	if index < 0 || index >= len(ml.items) {
		return nil
	}
	item := &ml.items[index]
	if item.separator {
		date := item.timestamp.Time().In(time.Local).Format(ml.cfg.DateSeparator.Format)
		return dateSeparator{date: date, fill: ml.cfg.DateSeparator.Character, style: ml.cfg.Theme.MessagesList.MessageStyle.Style.Dim(true)}
	}

	if item.view == nil {
		message := item.message
		view := &messageView{render: func(width int) richtext.Text {
			ml.renderWidth = width
			return ml.renderMessage(message, ml.cfg.Theme.MessagesList.MessageStyle.Style)
		}}
		for _, s := range previewSources(message) {
			if preview, ok := item.previews[s.proxy]; ok {
				view.images = append(view.images, preview)
			}
		}
		item.view = view
	}
	return item.view
}

// messageView renders a message followed by its image previews, caching the wrapped lines until the width changes.
type messageView struct {
	render   func(width int) richtext.Text
	images   []tviewimage.Widget
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

func (v *messageView) Rows(width int) int {
	v.at(width)
	rows := len(v.lines)
	for _, preview := range v.images {
		_, height := preview.Size()
		rows += height.Cells()
	}
	return rows
}

func (v *messageView) Draw(screen tview.Screen, area tview.Rectangle) {
	v.at(area.Width).Draw(screen, tview.Rectangle{X: area.X, Y: area.Y, Width: area.Width, Height: len(v.lines)})
	y := area.Y + len(v.lines)
	for _, preview := range v.images {
		_, height := preview.Size()
		preview.Draw(screen, tview.Rectangle{X: area.X, Y: y, Width: area.Width, Height: height.Cells()})
		y += height.Cells()
	}
}

func (v *messageView) Handle(msg tview.Msg, area tview.Rectangle) tview.Msg { return msg }

// dateSeparator is a line with the date in the middle, filled to the width it is drawn at.
type dateSeparator struct {
	date, fill string
	style      tcell.Style
}

func (dateSeparator) Rows(int) int { return 1 }

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

func (ml *Model) renderMessage(message discord.Message, baseStyle tcell.Style) richtext.Text {
	builder := new(richtext.Builder)
	ml.writeMessage(builder, message, baseStyle)
	return builder.Finish()
}

// rebuildItems replaces date separators while retaining message views and selection.
func (ml *Model) rebuildItems() {
	items := make([]messageItem, 0, len(ml.items))
	cursor, selected := ml.cursor(), -1
	var previous discord.Timestamp
	for i, item := range ml.items {
		if item.separator {
			continue
		}
		if ml.cfg.DateSeparator.Enabled && (len(items) == 0 || !sameLocalDate(previous, item.message.Timestamp)) {
			items = append(items, messageItem{separator: true, timestamp: item.message.Timestamp})
		}
		if i == cursor {
			selected = len(items)
		}
		items = append(items, item)
		previous = item.message.Timestamp
	}
	ml.items = items
	ml.setCursor(selected)
}

func sameLocalDate(a discord.Timestamp, b discord.Timestamp) bool {
	ta := a.Time().In(time.Local)
	tb := b.Time().In(time.Local)
	return ta.Year() == tb.Year() && ta.YearDay() == tb.YearDay()
}

// messageIndex finds the next message in direction (-1 or 1), excluding start.
func (ml *Model) messageIndex(start, direction int) int {
	for i := start + direction; i >= 0 && i < len(ml.items); i += direction {
		if !ml.items[i].separator {
			return i
		}
	}
	return -1
}

func (ml *Model) onRowCursorChanged(index int) {
	if index < 0 || index >= len(ml.items) || !ml.items[index].separator {
		return
	}
	target := ml.messageIndex(index, -1)
	if target == -1 {
		target = ml.messageIndex(index, 1)
	}
	ml.setCursor(target)
}

func (ml *Model) writeMessage(builder *richtext.Builder, message discord.Message, baseStyle tcell.Style) {
	if ml.cfg.HideBlockedUsers {
		isBlocked := ml.state.UserIsBlocked(message.Author.ID)
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
	ml.drawReactions(builder, message.Reactions, baseStyle)
}

func (ml *Model) drawReactions(builder *richtext.Builder, reactions []discord.Reaction, baseStyle tcell.Style) {
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
		style := ml.cfg.Theme.MessagesList.ReactionStyle.Style
		if reaction.Me {
			style = ml.cfg.Theme.MessagesList.OwnReactionStyle.Style
		}
		builder.Write(name+" "+strconv.Itoa(reaction.Count), tview.MergeStyle(baseStyle, style))
	}
}

func (ml *Model) formatTimestamp(ts discord.Timestamp) string {
	return ts.Time().In(time.Local).Format(ml.cfg.Timestamps.Format)
}

func (ml *Model) drawTimestamps(builder *richtext.Builder, ts discord.Timestamp, baseStyle tcell.Style) {
	dimStyle := baseStyle.Dim(true)
	builder.Write(ml.formatTimestamp(ts)+" ", dimStyle)
}

func (ml *Model) drawAuthor(builder *richtext.Builder, message discord.Message, baseStyle tcell.Style) {
	name := message.Author.DisplayOrUsername()
	foreground := tcell.ColorDefault

	if member := ml.memberForMessage(message); member != nil {
		if member.Nick != "" {
			name = member.Nick
		}

		color, ok := state.MemberColor(member, func(id discord.RoleID) *discord.Role {
			r, _ := ml.state.Cabinet.Role(message.GuildID, id)
			return r
		})
		if ok {
			foreground = tcell.NewHexColor(int32(color))
		}
	}

	style := baseStyle.Foreground(foreground).Bold(true)
	builder.Write(name+" ", style)
}

func (ml *Model) memberForMessage(message discord.Message) *discord.Member {
	// Webhooks do not have nicknames or roles.
	if !message.GuildID.IsValid() || message.WebhookID.IsValid() {
		return nil
	}

	member, err := ml.state.Cabinet.Member(message.GuildID, message.Author.ID)
	if err != nil {
		slog.Error("failed to get member from state", "guild_id", message.GuildID, "member_id", message.Author.ID, "err", err)
		return nil
	}
	return member
}

// drawContent renders the message body and returns the parsed markdown AST together with the source bytes it indexes into, so callers can reuse them instead of re-parsing the same content (see drawEmbeds).
// root is nil when markdown rendering is disabled.
func (ml *Model) drawContent(builder *richtext.Builder, message discord.Message, baseStyle tcell.Style) (ast.Node, []byte) {
	content, root, source := ml.renderContent(message, baseStyle, false)
	if ml.cfg.Markdown.Enabled && !builder.LineEmpty() {
		startsWithCodeBlock := false
		if root != nil {
			if first := root.FirstChild(); first != nil {
				_, startsWithCodeBlock = first.(*ast.FencedCodeBlock)
			}
		}

		if startsWithCodeBlock {
			// Keep code blocks visually separate from "timestamp + author".
			builder.NewLine()
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

func (ml *Model) renderContent(message discord.Message, baseStyle tcell.Style, forceMarkdown bool) (richtext.Text, ast.Node, []byte) {
	// Keep one rendering path for both normal messages and embed fragments so we preserve mention/link parsing behavior consistently across both.
	if forceMarkdown || ml.cfg.Markdown.Enabled {
		c := []byte(message.Content)
		root := md.ParseWithMessage(c, *ml.state.Cabinet, &message)
		return ml.renderer.RenderText(c, root, baseStyle), root, c
	}

	b := new(richtext.Builder)
	b.Write(message.Content, baseStyle)
	return b.Finish(), nil, nil
}

func (ml *Model) drawSnapshotContent(builder *richtext.Builder, parent discord.Message, snapshot discord.MessageSnapshotMessage, baseStyle tcell.Style) {
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
	ml.drawContent(builder, message, baseStyle)
}

func (ml *Model) drawDefaultMessage(builder *richtext.Builder, message discord.Message, baseStyle tcell.Style) {
	if ml.cfg.Timestamps.Enabled {
		ml.drawTimestamps(builder, message.Timestamp, baseStyle)
	}

	ml.drawAuthor(builder, message, baseStyle)
	contentRoot, contentSource := ml.drawContent(builder, message, baseStyle)

	if message.EditedTimestamp.IsValid() {
		dimStyle := baseStyle.Dim(true)
		builder.Write(" (edited)", dimStyle)
	}

	ml.drawEmbeds(builder, message, baseStyle, contentRoot, contentSource)

	attachmentStyle := tview.MergeStyle(baseStyle, ml.cfg.Theme.MessagesList.AttachmentStyle.Style)
	for _, a := range message.Attachments {
		builder.NewLine()
		if ml.cfg.Attachments.ShowLinks {
			builder.Write(a.Filename+":", attachmentStyle)
			builder.NewLine()
			builder.Write(a.URL, attachmentStyle.Url(a.URL))
		} else {
			builder.Write(a.Filename, attachmentStyle)
		}
	}
}

func (ml *Model) drawEmbeds(builder *richtext.Builder, message discord.Message, baseStyle tcell.Style, contentRoot ast.Node, contentSource []byte) {
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

	lineStyles := embedLineStyles(baseStyle, ml.cfg.Theme.MessagesList.Embeds)
	defaultBarStyle := baseStyle.Dim(true)
	prefixText := "  ▎ "
	prefixWidth := uniseg.StringWidth(prefixText)
	// Wrap against the list's width, so the message is rendered again when it changes.
	wrapWidth := max(ml.renderWidth-prefixWidth, 1)

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
			rendered, _, _ := ml.renderContent(msg, lineStyle, line.Kind == embedLineDescription)
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

func embedLineStyles(baseStyle tcell.Style, theme config.MessagesListEmbedsTheme) [8]tcell.Style {
	styles := [8]tcell.Style{}
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

func (ml *Model) drawForwardedMessage(builder *richtext.Builder, message discord.Message, baseStyle tcell.Style) {
	dimStyle := baseStyle.Dim(true)
	ml.drawTimestamps(builder, message.Timestamp, baseStyle)
	ml.drawAuthor(builder, message, baseStyle)
	builder.Write(ml.cfg.Theme.MessagesList.ForwardedIndicator+" ", dimStyle)
	ml.drawSnapshotContent(builder, message, message.MessageSnapshots[0].Message, baseStyle)
	builder.Write(" ("+ml.formatTimestamp(message.MessageSnapshots[0].Message.Timestamp)+") ", dimStyle)
}

func (ml *Model) drawReplyMessage(builder *richtext.Builder, message discord.Message, baseStyle tcell.Style) {
	dimStyle := baseStyle.Dim(true)
	builder.Write(ml.cfg.Theme.MessagesList.ReplyIndicator+" ", dimStyle)

	if m := message.ReferencedMessage; m != nil {
		m.GuildID = message.GuildID
		ml.drawAuthor(builder, *m, dimStyle)
		ml.drawContent(builder, *m, dimStyle)
	} else {
		builder.Write("Original message was deleted", dimStyle)
	}

	builder.NewLine()
	ml.drawDefaultMessage(builder, message, baseStyle)
}

func (ml *Model) drawPinnedMessage(builder *richtext.Builder, message discord.Message, baseStyle tcell.Style) {
	builder.Write(message.Author.DisplayOrUsername(), baseStyle)
	builder.Write(" pinned a message.", baseStyle)
}

func (ml *Model) selectedMessage() (*discord.Message, bool) {
	cursor := ml.cursor()
	if cursor < 0 || cursor >= len(ml.items) || ml.items[cursor].separator {
		return nil, false
	}

	return &ml.items[cursor].message, true
}

func (ml Model) Update(msg tview.Msg) (Model, tview.Cmd) {
	cmd := ml.update(msg)
	return ml, cmd
}

// update changes ml in response to msg and returns a command to run, or nil.
func (ml *Model) update(msg tview.Msg) tview.Cmd {
	switch msg := msg.(type) {
	case tview.KeyMsg:
		switch {
		case keybind.Matches(msg, ml.cfg.Keybinds.MessagesList.Cancel.Keybind):
			ml.clearSelection()
			return nil
		case keybind.Matches(msg, ml.cfg.Keybinds.MessagesList.SelectUp.Keybind):
			return ml.selectUp()
		case keybind.Matches(msg, ml.cfg.Keybinds.MessagesList.SelectDown.Keybind):
			ml.selectDown()
			return nil
		case keybind.Matches(msg, ml.cfg.Keybinds.MessagesList.SelectTop.Keybind):
			ml.selectTop()
			return nil
		case keybind.Matches(msg, ml.cfg.Keybinds.MessagesList.SelectBottom.Keybind):
			ml.selectBottom()
			return nil
		case keybind.Matches(msg, ml.cfg.Keybinds.MessagesList.SelectReply.Keybind):
			ml.selectReply()
			return nil
		case keybind.Matches(msg, ml.cfg.Keybinds.MessagesList.YankID.Keybind):
			return ml.yankMessageID()
		case keybind.Matches(msg, ml.cfg.Keybinds.MessagesList.YankContent.Keybind):
			return ml.yankContent()
		case keybind.Matches(msg, ml.cfg.Keybinds.MessagesList.YankURL.Keybind):
			return ml.yankURL()
		case keybind.Matches(msg, ml.cfg.Keybinds.MessagesList.Open.Keybind):
			return ml.open()
		case keybind.Matches(msg, ml.cfg.Keybinds.MessagesList.OpenInBrowser.Keybind):
			return ml.openInBrowser()
		case keybind.Matches(msg, ml.cfg.Keybinds.MessagesList.Download.Keybind):
			return ml.download()
		case keybind.Matches(msg, ml.cfg.Keybinds.MessagesList.Reply.Keybind):
			return ml.reply(false)
		case keybind.Matches(msg, ml.cfg.Keybinds.MessagesList.ReplyMention.Keybind):
			return ml.reply(true)
		case keybind.Matches(msg, ml.cfg.Keybinds.MessagesList.Edit.Keybind):
			return ml.editSelectedMessage()
		case keybind.Matches(msg, ml.cfg.Keybinds.MessagesList.Delete.Keybind):
			return ml.deleteSelectedMessage()
		case keybind.Matches(msg, ml.cfg.Keybinds.MessagesList.DeleteConfirm.Keybind):
			return ml.confirmDelete()
		}
	case olderMessagesLoadedMsg:
		selectedChannel := ml.channel
		if selectedChannel == nil || selectedChannel.ID != msg.ChannelID {
			return nil
		}
		prevCursor := ml.cursor()

		older := make([]messageItem, len(msg.Older))
		for i, message := range msg.Older {
			older[i] = messageItem{message: message}
		}
		first := ml.messageIndex(-1, 1)
		ml.items = slices.Concat(older, ml.items)
		switch {
		case prevCursor == first && len(older) > 0:
			ml.setCursor(len(older) - 1)
		case prevCursor >= 0:
			ml.setCursor(prevCursor + len(older))
		}
		ml.rebuildItems()
		if selectedChannel.GuildID.IsValid() {
			return tview.Batch(ml.requestGuildMembers(selectedChannel.GuildID, msg.Older), ml.loadPreviews())
		}
		return ml.loadPreviews()
	case previewLoadedMsg:
		index := slices.IndexFunc(ml.items, func(item messageItem) bool { return !item.separator && item.message.ID == msg.messageID })
		if index < 0 || ml.items[index].previews == nil {
			return nil
		}
		preview, cmd := msg.preview, tview.Cmd(nil)
		if ml.kitty {
			ml.kittyImages++
			preview = preview.Kitty(ml.kittyImages%63488 + 1)
			cmd = preview.Transmit()
		}
		ml.items[index].previews[msg.proxy] = preview
		// Invalidate the view so it is laid out with the preview.
		ml.items[index].view = nil
		return cmd
	case tview.TerminalInfoMsg:
		switch ml.cfg.Attachments.Protocol {
		case "kitty":
			ml.kitty = true
		case "halfblocks":
			ml.kitty = false
		default:
			// WezTerm and Konsole support kitty graphics but not Unicode placeholders.
			ml.kitty = msg.Name == "kitty" || msg.Name == "ghostty"
		}
	case listMsg:
		ml.selectionState.Apply(list.Change(msg))
		ml.onRowCursorChanged(ml.cursor())
	}
	return nil
}

func (ml *Model) selectDown() {
	if ml.cursor() == -1 {
		ml.selectBottom()
	} else if next := ml.messageIndex(ml.cursor(), 1); next >= 0 {
		ml.setCursor(next)
	}
}

func (ml *Model) selectTop() {
	ml.setCursor(ml.messageIndex(-1, 1))
}

func (ml *Model) selectBottom() {
	ml.setCursor(ml.messageIndex(len(ml.items), -1))
}

func (ml *Model) selectReply() {
	messages := ml.items
	cursor := ml.cursor()
	if cursor < 0 || cursor >= len(messages) {
		return
	}

	if ref := messages[cursor].message.ReferencedMessage; ref != nil {
		refIdx := slices.IndexFunc(messages, func(m messageItem) bool {
			return !m.separator && m.message.ID == ref.ID
		})
		if refIdx != -1 {
			ml.setCursor(refIdx)
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

func messageURLs(msg discord.Message) []string {
	combined := slices.Concat(extractURLs(msg.Content), extractEmbedURLs(msg.Embeds))

	urls := make([]string, 0, len(combined))
	seen := make(map[string]struct{}, len(combined))
	for _, u := range combined {
		u = strings.TrimSpace(u)
		if u == "" {
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

func attachmentErr(what string, err error) tview.Msg {
	slog.Error("failed to "+what, "err", err)
	return ui.ModalMsg{Text: "Failed to " + what + ": " + err.Error(), Buttons: []ui.ModalButton{{Label: "OK"}}}
}

func downloadAttachment(attachment discord.Attachment, destination string) error {
	resp, err := http.Get(attachment.URL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected HTTP status: %s", resp.Status)
	}
	file, err := os.Create(destination)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = io.Copy(file, resp.Body)
	return err
}

// SelectLastOwn selects the last message in the selected channel that the user can edit, and returns it.
func (ml *Model) SelectLastOwn() (discord.Message, bool) {
	channel := ml.channel
	if channel == nil {
		return discord.Message{}, false
	}
	for i, item := range slices.Backward(ml.items) {
		message := item.message
		if item.separator || message.ChannelID != channel.ID || !ui.IsMe(ml.state, message.Author.ID) ||
			!message.ID.IsValid() || message.Content == "" || len(message.MessageSnapshots) > 0 ||
			(message.Type != discord.DefaultMessage && message.Type != discord.InlinedReplyMessage) {
			continue
		}
		ml.setCursor(i)
		return message, true
	}
	return discord.Message{}, false
}

func (ml *Model) canDeleteMessage(message discord.Message) bool {
	return ui.IsMe(ml.state, message.Author.ID) ||
		(message.GuildID.IsValid() && ml.state.HasPermissions(message.ChannelID, discord.PermissionManageMessages))
}

func (ml *Model) InvalidateRendered() {
	for i := range ml.items {
		ml.items[i].view = nil
	}
}

func (ml Model) ShortHelp() []keybind.Keybind {
	cfg := ml.cfg.Keybinds.MessagesList
	help := []keybind.Keybind{
		cfg.SelectUp.Keybind,
		cfg.SelectDown.Keybind,
		cfg.Cancel.Keybind,
	}

	if selectedMessage, ok := ml.selectedMessage(); ok {
		if !ui.IsMe(ml.state, selectedMessage.Author.ID) {
			help = append(help, cfg.Reply.Keybind)
		}
		if len(selectedMessage.Attachments) != 0 || len(messageURLs(*selectedMessage)) != 0 {
			help = append(help, cfg.Open.Keybind)
		}
	}

	return help
}

func (ml Model) FullHelp() [][]keybind.Keybind {
	cfg := ml.cfg.Keybinds.MessagesList
	var actions, manage, attachments []keybind.Keybind
	if message, ok := ml.selectedMessage(); ok {
		mine := ui.IsMe(ml.state, message.Author.ID)
		if !mine {
			actions = append(actions, cfg.Reply.Keybind, cfg.ReplyMention.Keybind)
		}
		if message.ReferencedMessage != nil {
			actions = append(actions, cfg.SelectReply.Keybind)
		}
		if mine {
			manage = append(manage, cfg.Edit.Keybind)
		}
		if ml.canDeleteMessage(*message) {
			manage = append(manage, cfg.DeleteConfirm.Keybind)
			if len(cfg.Delete.Keys()) != 0 {
				manage = append(manage, cfg.Delete.Keybind)
			}
		}
		if len(message.Attachments) != 0 || len(messageURLs(*message)) != 0 {
			attachments = append(attachments, cfg.Open.Keybind, cfg.OpenInBrowser.Keybind)
		}
		if len(message.Attachments) != 0 {
			attachments = append(attachments, cfg.Download.Keybind)
		}
	}
	actions = append(actions, cfg.Cancel.Keybind)

	return [][]keybind.Keybind{
		{cfg.SelectUp.Keybind, cfg.SelectDown.Keybind, cfg.SelectTop.Keybind, cfg.SelectBottom.Keybind},
		{cfg.ScrollUp.Keybind, cfg.ScrollDown.Keybind, cfg.ScrollTop.Keybind, cfg.ScrollBottom.Keybind},
		actions,
		manage,
		attachments,
		{cfg.YankContent.Keybind, cfg.YankURL.Keybind, cfg.YankID.Keybind},
	}
}
