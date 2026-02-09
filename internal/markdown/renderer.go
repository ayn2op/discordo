package markdown

import (
	"strconv"
	"strings"

	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/discordo/internal/ui"
	"github.com/ayn2op/tview"
	"github.com/diamondburned/ningen/v3/discordmd"
	"github.com/gdamore/tcell/v3"
	"github.com/yuin/goldmark/ast"
	"github.com/rivo/uniseg"
)

type Renderer struct {
	theme config.MessagesListTheme

	listIx     *int
	listNested int

	spoil bool
}

func NewRenderer(theme config.MessagesListTheme) *Renderer {
	return &Renderer{theme: theme}
}

func (r *Renderer) RenderLines(source []byte, node ast.Node, base tcell.Style, showSpoiler bool) []tview.Line {
	r.listIx = nil
	r.listNested = 0
	r.spoil = false

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
				builder.Write(r.checkAndSpoil(string(node.Segment.Value(source))), currentStyle())
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
				if language := node.Language(source); language != nil {
					builder.Write("|=> "+r.checkAndSpoil(string(language)), currentStyle())
					builder.NewLine()
				}

				lines := node.Lines()
				for i := range lines.Len() {
					line := lines.At(i)
					builder.Write("| "+r.checkAndSpoil(string(line.Value(source))), currentStyle())
				}
			}
		case *ast.AutoLink:
			if entering {
				style := currentStyle()
				if !r.spoil {
					style = ui.MergeStyle(style, r.theme.URLStyle.Style)
				}
				builder.Write(r.checkAndSpoil(string(node.URL(source))), style)
			}
		case *ast.Link:
			if !r.spoil {
				if entering {
					pushStyle(ui.MergeStyle(currentStyle(), r.theme.URLStyle.Style))
				} else {
					popStyle()
				}
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
			if node.Attr.Has(discordmd.AttrSpoiler) && !showSpoiler {
				r.spoil = entering
			} else {
				if entering {
					pushStyle(r.applyInlineAttr(currentStyle(), node.Attr))
				} else {
					popStyle()
				}
			}
		case *discordmd.Mention:
			if entering {
				style := currentStyle()
				if !r.spoil {
					style = ui.MergeStyle(style, r.theme.MentionStyle.Style)
					style = style.Bold(true)
				}
				builder.Write(r.checkAndSpoil(mentionText(node)), style)
			}
		case *discordmd.Emoji:
			if entering {
				style := currentStyle()
				if !r.spoil {
					style = ui.MergeStyle(style, r.theme.EmojiStyle.Style)
				}
				builder.Write(r.checkAndSpoil(":"+node.Name+":"), style)
			}
		}
		return ast.WalkContinue, nil
	})

	return builder.Finish()
}

func (r *Renderer) applyInlineAttr(style tcell.Style, attr discordmd.Attribute) tcell.Style {
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
	case discordmd.AttrSpoiler:
		return r.theme.ShownSpoilerStyle.Style
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

func (r *Renderer) checkAndSpoil(s string) string {
	if r.spoil {
		return strings.Repeat(r.theme.SpoilCharacter, uniseg.StringWidth(s))
	}
	return s
}
