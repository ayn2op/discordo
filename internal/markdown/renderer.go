package markdown

import (
	"strconv"
	"strings"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	md "github.com/ayn2op/arikawa/v3/markdown"
	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/tview"
	"github.com/ayn2op/tview/text"
	"github.com/gdamore/tcell/v3"
	"github.com/yuin/goldmark/v2/ast"
)

type Renderer struct {
	cfg *config.Config
}

const codeBlockIndent = "    "

func NewRenderer(cfg *config.Config) *Renderer {
	return &Renderer{cfg: cfg}
}

// RenderText returns the spans of the markdown node. emoji returns the image of the custom emoji with an ID as a line of spans, or nil if it has none.
func (r *Renderer) RenderText(source []byte, node ast.Node, base tview.Style, emoji func(id string) []text.Span) []text.Span {
	builder := new(Builder)
	styleStack := []tview.Style{base}
	linkDepth := 0
	listDepth := 0

	currentStyle := func() tview.Style {
		return styleStack[len(styleStack)-1]
	}
	pushStyle := func(style tview.Style) {
		styleStack = append(styleStack, style)
	}
	popStyle := func() {
		if len(styleStack) > 1 {
			styleStack = styleStack[:len(styleStack)-1]
		}
	}

	theme := r.cfg.Theme.MessagesList
	_ = ast.Walk(node, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			switch node := node.(type) {
			case *ast.Heading:
				builder.NewLine()
			case *ast.ListItem:
				// A list that the item ends with already ended the line.
				if _, ok := node.LastChild().(*ast.List); !ok {
					builder.NewLine()
				}
			case *ast.Link:
				linkDepth--
				popStyle()
			case *ast.List:
				listDepth--
			case *md.Inline:
				if !r.shouldMask(node) {
					popStyle()
				}
			}
			return ast.WalkContinue, nil
		}

		switch node := node.(type) {
		case *ast.Heading:
			builder.Write(strings.Repeat("#", node.Level)+" ", currentStyle())
		case *ast.Text:
			builder.Write(string(node.Value.Bytes(source)), currentStyle())
			switch {
			case node.HardLineBreak():
				builder.NewLine()
				builder.NewLine()
			case node.SoftLineBreak():
				builder.NewLine()
			}
		case *ast.CodeBlock:
			builder.NewLine()
			r.renderCodeBlock(builder, source, node, currentStyle())
		case *ast.AutoLink:
			url := string(node.Destination.Bytes(source))
			style := tview.MergeStyle(currentStyle(), theme.URLStyle.Style).Url(url)
			builder.Write(url, style)
		case *ast.Link:
			url := string(node.Destination.Bytes(source))
			linkDepth++
			pushStyle(tview.MergeStyle(currentStyle(), theme.URLStyle.Style).Url(url))
		case *ast.List:
			builder.NewLine()
			listDepth++
		case *ast.ListItem:
			builder.Write(strings.Repeat("  ", listDepth-1)+listMarker(node), currentStyle())
		case *md.Inline:
			if r.shouldMask(node) {
				builder.Write("[spoiler]", currentStyle())
				return ast.WalkSkipChildren, nil
			}
			pushStyle(applyInlineAttr(currentStyle(), node.Attr, linkDepth > 0))
		case *md.Mention:
			builder.Write(mentionText(node), tview.MergeStyle(currentStyle(), theme.MentionStyle.Style))
		case *md.Emoji:
			cfg := r.cfg.UI.MessagesList.Emojis
			if image := emoji(node.ID); image != nil {
				builder.WriteSpans(image...)
				if !cfg.ShowEmojiName {
					return ast.WalkContinue, nil
				}

				builder.Write(" ", currentStyle())
			}
			builder.Write(cfg.Name(node.Name), tview.MergeStyle(currentStyle(), theme.EmojiStyle.Style))
		}
		return ast.WalkContinue, nil
	})

	return builder.Finish()
}

func (r *Renderer) renderCodeBlock(builder *Builder, source []byte, node *ast.CodeBlock, base tview.Style) {
	var code strings.Builder
	lines := node.Value.Segments()
	for _, line := range lines {
		code.Write(line.Bytes(source))
	}

	language, _ := node.Language(source)
	language = strings.TrimSpace(language)
	lexer := lexers.Get(language)
	declaredLanguageSupported := lexer != nil

	// Detect the language from its content.
	var analyzed bool
	if lexer == nil {
		lexer = lexers.Analyse(code.String())
		analyzed = lexer != nil
	}
	if lexer == nil {
		lexer = lexers.Fallback
	}

	// Some lexers emit many tiny tokens; coalesce runs of the same type.
	lexer = chroma.Coalesce(lexer)

	var header string
	switch {
	case analyzed:
		header = "code: analyzed"
	case language == "":
		header = "code"
	case !declaredLanguageSupported:
		header = "code: " + language
	}
	if header != "" {
		builder.Write(codeBlockIndent+header, base.Dim(true))
		builder.NewLine()
	}

	iterator, err := lexer.Tokenise(nil, code.String())
	if err != nil {
		for _, line := range lines {
			builder.Write(codeBlockIndent+string(line.Bytes(source)), base)
		}
		return
	}

	theme := styles.Get(r.cfg.Markdown.Theme)
	if theme == nil {
		theme = styles.Fallback
	}

	builder.Write(codeBlockIndent, base)
	for token := iterator(); token != chroma.EOF; token = iterator() {
		style := applyChromaStyle(base, theme.Get(token.Type))
		// Chroma tokens may include embedded newlines, so split and re-emit with indentation on each visual line.
		parts := strings.Split(token.Value, "\n")
		for i, part := range parts {
			if i > 0 {
				builder.NewLine()
				builder.Write(codeBlockIndent, base)
			}
			if part != "" {
				builder.Write(part, style)
			}
		}
	}
}

// shouldMask reports whether node is a spoiler shown as "[spoiler]" in place of what it hides.
func (r *Renderer) shouldMask(node *md.Inline) bool {
	return r.cfg.Markdown.MaskSpoilers && node.Attr&md.AttrSpoiler != 0
}

// listMarker returns what goes before item: its number in an ordered list, counted from the list's start, or a dash.
func listMarker(item *ast.ListItem) string {
	list := item.Parent().(*ast.List)
	if !list.IsOrdered() {
		return "- "
	}

	number := list.Start
	for sibling := item.PreviousSibling(); sibling != nil; sibling = sibling.PreviousSibling() {
		number++
	}
	return strconv.Itoa(number) + ". "
}

func applyChromaStyle(base tview.Style, entry chroma.StyleEntry) tview.Style {
	style := base
	if entry.Colour.IsSet() {
		style = style.Foreground(tcell.NewRGBColor(
			int32(entry.Colour.Red()),
			int32(entry.Colour.Green()),
			int32(entry.Colour.Blue()),
		))
	}
	// Token background colors are not applied, so code blocks keep the user's terminal/chat background.

	// An attribute chroma passes on is left as it is.
	if entry.Bold != chroma.Pass {
		style = style.Bold(entry.Bold == chroma.Yes)
	}
	if entry.Italic != chroma.Pass {
		style = style.Italic(entry.Italic == chroma.Yes)
	}
	if entry.Underline != chroma.Pass {
		style = style.Underline(entry.Underline == chroma.Yes)
	}
	return style
}

func mentionText(node *md.Mention) string {
	switch {
	case node.Channel != nil:
		return "#" + node.Channel.Name
	case node.GuildUser != nil:
		name := node.GuildUser.DisplayOrUsername()
		if member := node.GuildUser.Member; member != nil && member.Nick != "" {
			name = member.Nick
		}
		return "@" + name
	case node.GuildRole != nil:
		return "@" + node.GuildRole.Name
	default:
		return ""
	}
}

func applyInlineAttr(style tview.Style, attr md.Attribute, inLink bool) tview.Style {
	if attr&md.AttrBold != 0 {
		style = style.Bold(true)
	}
	if attr&md.AttrItalics != 0 {
		style = style.Italic(true)
	}
	if attr&md.AttrUnderline != 0 {
		style = style.Underline(true)
	}
	if attr&md.AttrStrikethrough != 0 {
		style = style.StrikeThrough(true)
	}
	if attr&md.AttrMonospace != 0 {
		// Avoid reverse-video inside links.
		// Link labels like `hash` should still look like links, not highlighted blocks.
		if !inLink {
			style = style.Reverse(true)
		}
	}
	return style
}
