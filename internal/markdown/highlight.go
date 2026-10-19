//go:build !no_syntax_highlighting

package markdown

import (
	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/ayn2op/tview"
	"github.com/gdamore/tcell/v3"
)

func (r *Renderer) writeCodeBlock(builder *Builder, code string, language string, base tview.Style) {
	lexer := lexers.Get(language)
	declaredLanguageSupported := lexer != nil

	// Detect the language from its content.
	var analyzed bool
	if lexer == nil {
		lexer = lexers.Analyse(code)
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

	iterator, err := lexer.Tokenise(nil, code)
	if err != nil {
		builder.Write(codeBlockIndent, base)
		writeCode(builder, code, base, base)
		return
	}

	theme := styles.Get(r.cfg.Markdown.Theme)
	if theme == nil {
		theme = styles.Fallback
	}

	builder.Write(codeBlockIndent, base)
	for token := iterator(); token != chroma.EOF; token = iterator() {
		style := applyChromaStyle(base, theme.Get(token.Type))
		writeCode(builder, token.Value, style, base)
	}
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
