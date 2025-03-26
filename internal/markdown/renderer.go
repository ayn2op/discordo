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
			io.WriteString(w, "\n")
		case *ast.FencedCodeBlock:
			io.WriteString(w, "\n")

			if entering {
				// body
				for i := range n.Lines().Len() {
					line := n.Lines().At(i)
					io.WriteString(w, "| ")
					w.Write(line.Value(source))
				}
			}
		case *ast.AutoLink:
			if entering {
				linkColor := r.config.Options["linkColor"].(string)
				io.WriteString(w, "["+linkColor+"]")
				w.Write(n.URL(source))
			} else {
				io.WriteString(w, "[-::]")
			}
		case *ast.Link:
			if entering {
				linkColor := r.config.Options["linkColor"].(string)
				io.WriteString(w, fmt.Sprintf("[%s:::%s]", linkColor, n.Destination))
			} else {
				io.WriteString(w, "[-:::-]")
			}
		case *ast.Text:
			if entering {
				w.Write(n.Segment.Value(source))
				switch {
				case n.HardLineBreak():
					io.WriteString(w, "\n\n")
				case n.SoftLineBreak():
					io.WriteString(w, "\n")
				}
			}

		case *discordmd.Inline:
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
				io.WriteString(w, "[::-]")
			}
		case *discordmd.Mention:
			if entering {
				io.WriteString(w, "[::b]")

				switch {
				case n.Channel != nil:
					io.WriteString(w, "#"+n.Channel.Name)
				case n.GuildUser != nil:
					io.WriteString(w, "@"+n.GuildUser.Username)
				case n.GuildRole != nil:
					io.WriteString(w, "@"+n.GuildRole.Name)
				}
			} else {
				io.WriteString(w, "[::-]")
			}
		case *discordmd.Emoji:
			if entering {
				emojiColor := r.config.Options["emojiColor"].(string)
				io.WriteString(w, "["+emojiColor+"]")
				io.WriteString(w, ":"+n.Name+":")
			} else {
				io.WriteString(w, "[-]")
			}
		}

		return ast.WalkContinue, nil
	})
}
