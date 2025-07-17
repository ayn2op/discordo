// Package markdown defines a renderer for tview style tags.
package markdown

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/tview"
	"github.com/diamondburned/ningen/v3/discordmd"
	"github.com/gdamore/tcell/v2"
	"github.com/yuin/goldmark/ast"
	gmr "github.com/yuin/goldmark/renderer"
)

var DefaultRenderer = newRenderer()

type renderer struct {
	config *gmr.Config

	listIx     *int
	listNested int
}

func newRenderer() *renderer {
	config := gmr.NewConfig()
	return &renderer{config: config}
}

// AddOptions implements renderer.Renderer.
func (r *renderer) AddOptions(opts ...gmr.Option) {
	for _, opt := range opts {
		opt.SetConfig(r.config)
	}
}

func (r *renderer) Render(w io.Writer, source []byte, node ast.Node) error {
	theme := r.config.Options["theme"].(config.Theme)
	return ast.Walk(node, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		switch node := node.(type) {
		case *ast.Document:
		// noop
		case *ast.Heading:
			r.renderHeading(w, node, entering)
		case *ast.Text:
			r.renderText(w, node, entering, source)
		case *ast.FencedCodeBlock:
			r.renderFencedCodeBlock(w, node, entering, source)
		case *ast.AutoLink:
			r.renderAutoLink(w, node, entering, source, theme.MessagesList.URLStyle.Style)
		case *ast.Link:
			r.renderLink(w, node, entering, theme.MessagesList.URLStyle.Style)
		case *ast.List:
			r.renderList(w, node, entering)
		case *ast.ListItem:
			r.renderListItem(w, entering)

		case *discordmd.Inline:
			r.renderInline(w, node, entering)
		case *discordmd.Mention:
			r.renderMention(w, node, entering, theme.MessagesList.MentionStyle.Style)
		case *discordmd.Emoji:
			r.renderEmoji(w, node, entering, theme.MessagesList.EmojiStyle.Style)
		}

		return ast.WalkContinue, nil
	})
}

func (r *renderer) renderHeading(w io.Writer, node *ast.Heading, entering bool) {
	if entering {
		io.WriteString(w, strings.Repeat("#", node.Level))
		io.WriteString(w, " ")
	} else {
		io.WriteString(w, tview.NewLine)
	}
}

func (r *renderer) renderFencedCodeBlock(w io.Writer, node *ast.FencedCodeBlock, entering bool, source []byte) {
	io.WriteString(w, tview.NewLine)

	if entering {
		// language
		if l := node.Language(source); l != nil {
			io.WriteString(w, "|=> ")
			w.Write(l)
			io.WriteString(w, tview.NewLine)
		}

		// body
		lines := node.Lines()
		for i := range lines.Len() {
			line := lines.At(i)
			io.WriteString(w, "| ")
			w.Write(line.Value(source))
		}
	}
}

func (r *renderer) renderAutoLink(w io.Writer, node *ast.AutoLink, entering bool, source []byte, urlStyle tcell.Style) {
	if entering {
		fg, bg, _ := urlStyle.Decompose()
		_, _ = fmt.Fprintf(w, "[%s:%s]", fg, bg)
		w.Write(node.URL(source))
	} else {
		io.WriteString(w, "[-:-]")
	}
}

func (r *renderer) renderLink(w io.Writer, node *ast.Link, entering bool, urlStyle tcell.Style) {
	if entering {
		fg, bg, _ := urlStyle.Decompose()
		_, _ = fmt.Fprintf(w, "[%s:%s::%s]", fg, bg, node.Destination)
	} else {
		io.WriteString(w, "[-:-::-]")
	}
}

func (r *renderer) renderList(w io.Writer, node *ast.List, entering bool) {
	if node.IsOrdered() {
		r.listIx = &node.Start
	} else {
		r.listIx = nil
	}

	if entering {
		io.WriteString(w, tview.NewLine)
		r.listNested++
	} else {
		r.listNested--
	}
}

func (r *renderer) renderListItem(w io.Writer, entering bool) {
	if entering {
		io.WriteString(w, strings.Repeat("  ", r.listNested-1))

		if r.listIx != nil {
			io.WriteString(w, strconv.Itoa(*r.listIx))
			io.WriteString(w, ". ")
			*r.listIx++
		} else {
			io.WriteString(w, "- ")
		}
	} else {
		io.WriteString(w, tview.NewLine)
	}
}

func (r *renderer) renderText(w io.Writer, node *ast.Text, entering bool, source []byte) {
	if entering {
		w.Write(node.Segment.Value(source))
		switch {
		case node.HardLineBreak():
			io.WriteString(w, strings.Repeat(tview.NewLine, 2))
		case node.SoftLineBreak():
			io.WriteString(w, tview.NewLine)
		}
	}
}

func (r *renderer) renderInline(w io.Writer, node *discordmd.Inline, entering bool) {
	var start, end string
	if entering {
		switch node.Attr {
		case discordmd.AttrBold:
			start = "[::b]"
			end = "[::B]"
		case discordmd.AttrItalics:
			start = "[::i]"
			end = "[::I]"
		case discordmd.AttrUnderline:
			start = "[::u]"
			end = "[::U]"
		case discordmd.AttrStrikethrough:
			start = "[::s]"
			end = "[::S]"
		case discordmd.AttrMonospace:
			start = "[::r]"
			end = "[::R]"
		}

		io.WriteString(w, start)
	} else {
		io.WriteString(w, end)
	}
}

func (r *renderer) renderMention(w io.Writer, node *discordmd.Mention, entering bool, style tcell.Style) {
	if entering {
		fg, bg, _ := style.Decompose()
		_, _ = fmt.Fprintf(w, "[%s:%s:b]", fg, bg)

		switch {
		case node.Channel != nil:
			io.WriteString(w, "#"+node.Channel.Name)
		case node.GuildUser != nil:
			name := node.GuildUser.DisplayOrUsername()
			if member := node.GuildUser.Member; member != nil && member.Nick != "" {
				name = member.Nick
			}

			io.WriteString(w, "@"+name)
		case node.GuildRole != nil:
			io.WriteString(w, "@"+node.GuildRole.Name)
		}
	} else {
		io.WriteString(w, "[-:-:B]")
	}
}

func (r *renderer) renderEmoji(w io.Writer, node *discordmd.Emoji, entering bool, emojiStyle tcell.Style) {
	if entering {
		fg, bg, _ := emojiStyle.Decompose()
		fmt.Fprintf(w, "[%s:%s]", fg, bg)
		io.WriteString(w, ":"+node.Name+":")
	} else {
		io.WriteString(w, "[-:-]")
	}
}
