package markdown

import (
	"strconv"
	"strings"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/discordo/internal/ui"
	"github.com/ayn2op/tview"
	"github.com/diamondburned/ningen/v3/discordmd"
	"github.com/gdamore/tcell/v3"
	"github.com/yuin/goldmark/ast"
)

type Renderer struct {
	cfg *config.Config

	listIx     *int
	listNested int
}

const codeBlockIndent = "    "

func NewRenderer(cfg *config.Config) *Renderer {
	return &Renderer{cfg: cfg}
}

func (r *Renderer) RenderLines(source []byte, node ast.Node, base tcell.Style) []tview.Line {
	r.listIx = nil
	r.listNested = 0

	builder := tview.NewLineBuilder()
	styleStack := []tcell.Style{base}

	currentStyle := func() tcell.Style {
		return styleStack[len(styleStack)-1]
	}
	pushStyle := func(style tcell.Style) {
		styleStack = append(styleStack, style)
	}
	popStyle := func() {
		if len(styleStack) > 1 {
			styleStack = styleStack[:len(styleStack)-1]
		}
	}

	theme := r.cfg.Theme.MessagesList
	_ = ast.Walk(node, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		switch node := node.(type) {
		case *ast.Document:
			// noop
		case *ast.Heading:
			if entering {
				builder.Write(strings.Repeat("#", node.Level)+" ", currentStyle())
			} else {
				builder.NewLine()
			}
		case *ast.Text:
			if entering {
				builder.Write(string(node.Segment.Value(source)), currentStyle())
				switch {
				case node.HardLineBreak():
					builder.NewLine()
					builder.NewLine()
				case node.SoftLineBreak():
					builder.NewLine()
				}
			}
		case *ast.FencedCodeBlock:
			if entering {
				builder.NewLine()
				r.renderFencedCodeBlock(builder, source, node, currentStyle())
			}
		case *ast.AutoLink:
			if entering {
				builder.Write(string(node.URL(source)), ui.MergeStyle(currentStyle(), theme.URLStyle.Style))
			}
		case *ast.Link:
			if entering {
				pushStyle(ui.MergeStyle(currentStyle(), theme.URLStyle.Style))
			} else {
				popStyle()
			}
		case *ast.List:
			if node.IsOrdered() {
				start := node.Start
				r.listIx = &start
			} else {
				r.listIx = nil
			}

			if entering {
				builder.NewLine()
				r.listNested++
			} else {
				r.listNested--
			}
		case *ast.ListItem:
			if entering {
				builder.Write(strings.Repeat("  ", r.listNested-1), currentStyle())
				if r.listIx != nil {
					builder.Write(strconv.Itoa(*r.listIx)+". ", currentStyle())
					*r.listIx++
				} else {
					builder.Write("- ", currentStyle())
				}
			} else {
				builder.NewLine()
			}
		case *discordmd.Inline:
			if entering {
				pushStyle(applyInlineAttr(currentStyle(), node.Attr))
			} else {
				popStyle()
			}
		case *discordmd.Mention:
			if entering {
				builder.Write(mentionText(node), ui.MergeStyle(currentStyle(), theme.MentionStyle.Style))
			}
		case *discordmd.Emoji:
			if entering {
				builder.Write(":"+node.Name+":", ui.MergeStyle(currentStyle(), theme.EmojiStyle.Style))
			}
		}
		return ast.WalkContinue, nil
	})

	return builder.Finish()
}

func (r *Renderer) renderFencedCodeBlock(builder *tview.LineBuilder, source []byte, node *ast.FencedCodeBlock, base tcell.Style) {
	var code strings.Builder
	lines := node.Lines()
	for i := range lines.Len() {
		line := lines.At(i)
		code.Write(line.Value(source))
	}

	language := strings.TrimSpace(string(node.Language(source)))
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

	// At this point, it should be noted that some lexers can be extremely chatty.
	// To mitigate this, use the coalescing lexer to coalesce runs of identical token types into a single token.
	lexer = chroma.Coalesce(lexer)

	// Show a fallback header when the language is omitted or unknown.
	headerStyle := base.Dim(true)
	if analyzed {
		builder.Write(codeBlockIndent+"code: analyzed", headerStyle)
		builder.NewLine()
	} else if language == "" {
		builder.Write(codeBlockIndent+"code", headerStyle)
		builder.NewLine()
	} else if !declaredLanguageSupported {
		builder.Write(codeBlockIndent+"code: "+language, headerStyle)
		builder.NewLine()
	}

	iterator, err := lexer.Tokenise(nil, code.String())
	if err != nil {
		for i := range lines.Len() {
			line := lines.At(i)
			builder.Write(codeBlockIndent+string(line.Value(source)), base)
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

func applyChromaStyle(base tcell.Style, entry chroma.StyleEntry) tcell.Style {
	style := base
	if entry.Colour.IsSet() {
		style = style.Foreground(tcell.NewRGBColor(
			int32(entry.Colour.Red()),
			int32(entry.Colour.Green()),
			int32(entry.Colour.Blue()),
		))
	}
	// Intentionally do not apply token background colors so code blocks keep the user's terminal/chat background.
	// if entry.Background.IsSet() {
	// 	style = style.Background(tcell.NewRGBColor(
	// 		int32(entry.Background.Red()),
	// 		int32(entry.Background.Green()),
	// 		int32(entry.Background.Blue()),
	// 	))
	// }
	switch entry.Bold {
	case chroma.Yes:
		style = style.Bold(true)
	case chroma.No:
		style = style.Bold(false)
	}
	switch entry.Italic {
	case chroma.Yes:
		style = style.Italic(true)
	case chroma.No:
		style = style.Italic(false)
	}
	switch entry.Underline {
	case chroma.Yes:
		style = style.Underline(true)
	case chroma.No:
		style = style.Underline(false)
	}
	return style
}

func mentionText(node *discordmd.Mention) string {
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

func applyInlineAttr(style tcell.Style, attr discordmd.Attribute) tcell.Style {
	switch attr {
	case discordmd.AttrBold:
		return style.Bold(true)
	case discordmd.AttrItalics:
		return style.Italic(true)
	case discordmd.AttrUnderline:
		return style.Underline(true)
	case discordmd.AttrStrikethrough:
		return style.StrikeThrough(true)
	case discordmd.AttrMonospace:
		return style.Reverse(true)
	}
	return style
}
