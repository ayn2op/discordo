package markdown

import (
	"fmt"
	"io"

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
			r.renderHeading(w)
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

func (r *renderer) renderHeading(w io.Writer) {
	io.WriteString(w, "\n")
}

func (r *renderer) renderFencedCodeBlock(w io.Writer, n *ast.FencedCodeBlock, entering bool, source []byte) {
	io.WriteString(w, "\n")

	if entering {
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
		linkColor := r.config.Options["linkColor"].(string)
		io.WriteString(w, "["+linkColor+"]")
		w.Write(n.URL(source))
	} else {
		io.WriteString(w, "[-::]")
	}
}

func (r *renderer) renderLink(w io.Writer, n *ast.Link, entering bool) {
	if entering {
		linkColor := r.config.Options["linkColor"].(string)
		io.WriteString(w, fmt.Sprintf("[%s:::%s]", linkColor, n.Destination))
	} else {
		io.WriteString(w, "[-:::-]")
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
		mentionColor := r.config.Options["mentionColor"].(string)
		_, _ = fmt.Fprintf(w, "[%s::b]", mentionColor)

		switch {
		case n.Channel != nil:
			io.WriteString(w, "#"+n.Channel.Name)
		case n.GuildUser != nil:
			username := n.GuildUser.DisplayOrUsername()
			if r.config.Options["showNicknames"].(bool) && n.GuildUser.Member != nil && n.GuildUser.Member.Nick != "" {
				username = n.GuildUser.Member.Nick
			}
			io.WriteString(w, "@"+username)
		case n.GuildRole != nil:
			io.WriteString(w, "@"+n.GuildRole.Name)
		}
	} else {
		io.WriteString(w, "[-::B]")
	}
}

func (r *renderer) renderEmoji(w io.Writer, n *discordmd.Emoji, entering bool) {
	if entering {
		emojiColor := r.config.Options["emojiColor"].(string)
		io.WriteString(w, "["+emojiColor+"]")
		io.WriteString(w, ":"+n.Name+":")
	} else {
		io.WriteString(w, "[-]")
	}
}
