package markdown

import (
	"strings"

	"github.com/ayn2op/tview"
	"github.com/diamondburned/ningen/v3/discordmd"
	"github.com/gdamore/tcell/v3"
	"github.com/yuin/goldmark/ast"
)

type InlineRenderer struct {}

func NewInlineRenderer() *InlineRenderer {
	return &InlineRenderer{}
}

func (r *InlineRenderer) RenderMarkdownLine(source []byte, base tcell.Style, replacer *strings.Replacer) tview.Line {
	node := discordmd.Parse(source)
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

	_ = ast.Walk(node, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		switch node := node.(type) {
		case *ast.Text:
			if entering {
				builder.Write(replacer.Replace(string(node.Segment.Value(source))), currentStyle())
			}
		case *discordmd.Inline:
			if entering {
				pushStyle(r.applyInlineAttr(currentStyle(), node.Attr))
			} else {
				popStyle()
			}
		}
		return ast.WalkContinue, nil
	})

	return builder.Finish()[0]
}

func (r *InlineRenderer) applyInlineAttr(style tcell.Style, attr discordmd.Attribute) tcell.Style {
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
