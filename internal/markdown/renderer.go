package markdown

import (
	"fmt"
	"io"
	"strings"

	"github.com/ayn2op/discordo/internal/config"
	"github.com/diamondburned/ningen/v3/discordmd"
	"github.com/yuin/goldmark/ast"
	gmr "github.com/yuin/goldmark/renderer"
)

var DefaultRenderer = newRenderer()

type renderer struct {
	config *gmr.Config
}

func newRenderer() *renderer {
	config := gmr.NewConfig()
	return &renderer{config}
}

// AddOptions implements renderer.Renderer.
func (r *renderer) AddOptions(opts ...gmr.Option) {
	for _, opt := range opts {
		opt.SetConfig(r.config)
	}
}

func (r *renderer) Render(w io.Writer, source []byte, n ast.Node) error {
	return ast.Walk(n, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		switch n := n.(type) {
		case *ast.Document:
		// noop
		case *ast.Heading:
			r.renderHeading(w, n, entering)
		case *ast.Text:
			r.renderText(w, n, entering, source)
		case *ast.FencedCodeBlock:
			r.renderFencedCodeBlock(w, n, entering, source)
		case *ast.AutoLink:
			r.renderAutoLink(w, n, entering, source)
		case *ast.Link:
			r.renderLink(w, n, entering)

		case *discordmd.Inline:
			r.renderInline(w, n, entering)
		case *discordmd.Mention:
			r.renderMention(w, n, entering)
		case *discordmd.Emoji:
			r.renderEmoji(w, n, entering)
		}

		return ast.WalkContinue, nil
	})
}

func (r *renderer) renderHeading(w io.Writer, n *ast.Heading, entering bool) {
	if entering {
		io.WriteString(w, strings.Repeat("#", n.Level))
		io.WriteString(w, " ")
	} else {
		io.WriteString(w, "\n")
	}
}

func (r *renderer) renderFencedCodeBlock(w io.Writer, n *ast.FencedCodeBlock, entering bool, source []byte) {
	io.WriteString(w, "\n")

	if entering {
		// language
		if l := n.Language(source); l != nil {
			io.WriteString(w, "|=> ")
			w.Write(l)
			io.WriteString(w, "\n")
		}

		// body
		lines := n.Lines()
		for i := range lines.Len() {
			line := lines.At(i)
			io.WriteString(w, "| ")
			w.Write(line.Value(source))
		}
	}
}

func (r *renderer) renderAutoLink(w io.Writer, n *ast.AutoLink, entering bool, source []byte) {
	if entering {
		theme := r.config.Options["theme"].(config.MessagesTextTheme)
		fg, bg, _ := theme.URLStyle.Decompose()
		_, _ = fmt.Fprintf(w, "[%s:%s]", fg, bg)
		w.Write(n.URL(source))
	} else {
		io.WriteString(w, "[-:-]")
	}
}

func (r *renderer) renderLink(w io.Writer, n *ast.Link, entering bool) {
	if entering {
		theme := r.config.Options["theme"].(config.MessagesTextTheme)
		fg, bg, _ := theme.URLStyle.Decompose()
		_, _ = fmt.Fprintf(w, "[%s:%s::%s]", fg, bg, n.Destination)
	} else {
		io.WriteString(w, "[-:-::-]")
	}
}

func (r *renderer) renderText(w io.Writer, n *ast.Text, entering bool, source []byte) {
	if entering {
		w.Write(n.Segment.Value(source))
		switch {
		case n.HardLineBreak():
			io.WriteString(w, "\n\n")
		case n.SoftLineBreak():
			io.WriteString(w, "\n")
		}
	}
}

func (r *renderer) renderInline(w io.Writer, n *discordmd.Inline, entering bool) {
	if entering {
		switch n.Attr {
		case discordmd.AttrBold:
			io.WriteString(w, "[::b]")
		case discordmd.AttrItalics:
			io.WriteString(w, "[::i]")
		case discordmd.AttrUnderline:
			io.WriteString(w, "[::u]")
		case discordmd.AttrStrikethrough:
			io.WriteString(w, "[::s]")
		case discordmd.AttrMonospace:
			io.WriteString(w, "[::r]")
		}
	} else {
		switch n.Attr {
		case discordmd.AttrBold:
			io.WriteString(w, "[::B]")
		case discordmd.AttrItalics:
			io.WriteString(w, "[::I]")
		case discordmd.AttrUnderline:
			io.WriteString(w, "[::U]")
		case discordmd.AttrStrikethrough:
			io.WriteString(w, "[::S]")
		case discordmd.AttrMonospace:
			io.WriteString(w, "[::R]")
		}
	}
}

func (r *renderer) renderMention(w io.Writer, n *discordmd.Mention, entering bool) {
	if entering {
		theme := r.config.Options["theme"].(config.MessagesTextTheme)
		fg, bg, _ := theme.MentionStyle.Decompose()
		_, _ = fmt.Fprintf(w, "[%s:%s:b]", fg, bg)

		switch {
		case n.Channel != nil:
			io.WriteString(w, "#"+n.Channel.Name)
		case n.GuildUser != nil:
			username := n.GuildUser.DisplayOrUsername()
			theme := r.config.Options["theme"].(config.MessagesTextTheme)
			if theme.ShowNicknames && n.GuildUser.Member != nil && n.GuildUser.Member.Nick != "" {
				username = n.GuildUser.Member.Nick
			}
			io.WriteString(w, "@"+username)
		case n.GuildRole != nil:
			io.WriteString(w, "@"+n.GuildRole.Name)
		}
	} else {
		io.WriteString(w, "[-:-:B]")
	}
}

func (r *renderer) renderEmoji(w io.Writer, n *discordmd.Emoji, entering bool) {
	if entering {
		theme := r.config.Options["theme"].(config.MessagesTextTheme)
		fg, bg, _ := theme.EmojiStyle.Decompose()
		fmt.Fprintf(w, "[%s:%s]", fg, bg)
		io.WriteString(w, ":"+n.Name+":")
	} else {
		io.WriteString(w, "[-:-]")
	}
}
