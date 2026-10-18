package messageslist

import (
	"net/url"
	"strings"

	"github.com/ayn2op/arikawa/v3/discord"
	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/discordo/internal/markdown"
	"github.com/ayn2op/tview"
	"github.com/ayn2op/tview/box"
	"github.com/ayn2op/tview/layout"
	"github.com/ayn2op/tview/text"
	"github.com/gdamore/tcell/v3"
	"github.com/yuin/goldmark/v2/ast"
)

const (
	// embedBar is drawn down the left of an embed, in the embed's color.
	embedBar = "▎"
	// embedIndent is the cells before the bar, and embedBarGap those between the bar and the text.
	embedIndent, embedBarGap = 2, 1
)

// embedViews returns each embed of message as its text with a bar down its left, or nil if the message does not show its embeds: one without any, by a blocked user, or that is not drawn with its own content, as a forward.
// The views capture the model and the message, so they are made once with the text of the message rather than in buildItem.
// root is the parsed content of message, or nil to parse it here.
func (m Model) embedViews(message discord.Message, root ast.Node) []tview.Widget {
	forward := message.Reference != nil && message.Reference.Type == discord.MessageReferenceTypeForward
	ownContent := message.Type == discord.DefaultMessage && !forward || message.Type == discord.InlinedReplyMessage
	if len(message.Embeds) == 0 || m.blocked(message.Author) || !ownContent {
		return nil
	}

	// Embed URLs are deduplicated against links already shown in the message body.
	contentListURLs := contentURLs(message.Content, root)
	seenURLs := make(map[string]struct{}, len(contentListURLs))
	for _, u := range contentListURLs {
		seenURLs[u] = struct{}{}
	}

	baseStyle := m.cfg.Theme.MessagesList.MessageStyle.Style
	lineStyles := embedLineStyles(baseStyle, m.cfg.Theme.MessagesList.Embeds)
	var views []tview.Widget
	for _, embed := range message.Embeds {
		lines := embedLines(embed, seenURLs)
		if len(lines) == 0 {
			continue
		}

		barStyle := baseStyle.Dim(true)
		if embed.Color != discord.NullColor && embed.Color != 0 {
			barStyle = barStyle.Foreground(tcell.NewHexColor(int32(embed.Color)))
		}

		content := &messageView{render: func(int) []text.Span { return m.renderEmbed(message, lines, lineStyles) }}
		bar := box.New(content).
			Borders(tview.BordersLeft).
			BorderSet(tview.BorderSet{Left: embedBar}).
			BorderStyle(barStyle).
			Padding(0, 0, embedBarGap, 0).
			Height(layout.Shrink)
		views = append(views, box.New(bar).Padding(0, 0, embedIndent, 0).Height(layout.Shrink))
	}

	return views
}

// renderEmbed returns lines of an embed of message, each on its own line in the style of its kind.
func (m Model) renderEmbed(message discord.Message, lines []embedLine, lineStyles [embedLineKinds]tview.Style) []text.Span {
	builder := new(markdown.Builder)
	for _, line := range lines {
		if !builder.LineEmpty() {
			builder.NewLine()
		}

		msg := message
		msg.Content = line.text
		// Embed descriptions are always markdown-rendered to match Discord's rich embed semantics, even when message markdown is globally disabled.
		rendered := m.renderContent(msg, nil, lineStyles[line.kind], line.kind == embedLineDescription)
		if line.url != "" {
			rendered = spansWithURL(rendered, line.url)
		}
		builder.WriteSpans(rendered...)
	}

	return builder.Finish()
}

func spansWithURL(spans []text.Span, rawURL string) []text.Span {
	out := make([]text.Span, len(spans))
	for i, span := range spans {
		out[i] = span
		out[i].Style = out[i].Style.Url(rawURL)
	}
	return out
}

type embedLine struct {
	text string
	kind embedLineKind
	url  string
}

type embedLineKind uint8

const (
	embedLineProvider embedLineKind = iota
	embedLineAuthor
	embedLineTitle
	embedLineDescription
	embedLineFieldName
	embedLineFieldValue
	embedLineFooter
	embedLineURL

	// embedLineKinds is how many kinds there are.
	embedLineKinds
)

func embedLineStyles(baseStyle tview.Style, theme config.MessagesListEmbedsThemeConfig) [embedLineKinds]tview.Style {
	return [embedLineKinds]tview.Style{
		embedLineProvider:    tview.MergeStyle(baseStyle, theme.ProviderStyle.Style),
		embedLineAuthor:      tview.MergeStyle(baseStyle, theme.AuthorStyle.Style),
		embedLineTitle:       tview.MergeStyle(baseStyle, theme.TitleStyle.Style),
		embedLineDescription: tview.MergeStyle(baseStyle, theme.DescriptionStyle.Style),
		embedLineFieldName:   tview.MergeStyle(baseStyle, theme.FieldNameStyle.Style),
		embedLineFieldValue:  tview.MergeStyle(baseStyle, theme.FieldValueStyle.Style),
		embedLineFooter:      tview.MergeStyle(baseStyle, theme.FooterStyle.Style),
		embedLineURL:         tview.MergeStyle(baseStyle, theme.URLStyle.Style),
	}
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
			text: s,
			kind: kind,
			url:  rawURL,
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
