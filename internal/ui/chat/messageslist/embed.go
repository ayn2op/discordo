package messageslist

import (
	"net/url"
	"strings"

	"github.com/ayn2op/arikawa/v3/discord"
	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/discordo/internal/markdown"
	"github.com/ayn2op/tview"
	"github.com/ayn2op/tview/text"
	"github.com/gdamore/tcell/v3"
	"github.com/rockorager/go-uucode"
	"github.com/yuin/goldmark/v2/ast"
)

// embedPrefix is drawn before each line of an embed, in the embed's color.
const embedPrefix = "  ▎ "

// embedPrefixWidth is how many cells embedPrefix takes.
var embedPrefixWidth = uucode.StringWidth(embedPrefix)

func (m Model) drawEmbeds(builder *markdown.Builder, message discord.Message, baseStyle tview.Style, width int, contentRoot ast.Node, contentSource []byte) {
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
	defaultPrefixStyle := baseStyle.Dim(true)
	// Wrap against the list's width, so the message is rendered again when it changes.
	wrapWidth := max(width-embedPrefixWidth, 1)

	for _, embed := range message.Embeds {
		lines := embedLines(embed, contentURLs)
		if len(lines) == 0 {
			continue
		}

		prefixStyle := defaultPrefixStyle
		if embed.Color != discord.NullColor && embed.Color != 0 {
			prefixStyle = prefixStyle.Foreground(tcell.NewHexColor(int32(embed.Color)))
		}
		prefix := text.Span{Text: embedPrefix, Style: prefixStyle}
		builder.NewLine()
		for _, line := range lines {
			if strings.TrimSpace(line.text) == "" {
				continue
			}
			msg := message
			msg.Content = line.text
			lineStyle := lineStyles[line.kind]
			// Embed descriptions are always markdown-rendered to match Discord's rich embed semantics, even when message markdown is globally disabled.
			rendered, _, _ := m.renderContent(msg, lineStyle, line.kind == embedLineDescription)
			if line.url != "" {
				rendered = spansWithURL(rendered, line.url)
			}

			// The prefix is added after wrapping, so every visual line has it, not only the first.
			for _, wrapped := range text.Lines(rendered, wrapWidth, text.WrappingGlyph) {
				if !builder.LineEmpty() {
					builder.NewLine()
				}

				builder.WriteSpans(prefix)
				builder.WriteSpans(wrapped...)
			}
		}
	}
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
