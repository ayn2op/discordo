package notifications

import (
	"io"

	"github.com/diamondburned/ningen/v3/discordmd"
	"github.com/yuin/goldmark/ast"
	gmr "github.com/yuin/goldmark/renderer"
)

// Using a modified version of the discordmd BasicRenderer
var defaultRenderer = newRenderer()

type renderer struct {
	config *gmr.Config
}

func newRenderer() *renderer {
	config := gmr.NewConfig()
	return &renderer{config}
}

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
		case *ast.Blockquote:
			io.WriteString(w, "\"")
		case *ast.Heading:
			io.WriteString(w, "\n")
		case *ast.FencedCodeBlock:
			io.WriteString(w, "\n")

			if entering {
				lines := n.Lines()
				for i := range lines.Len() {
					line := lines.At(i)
					io.WriteString(w, "| ")
					w.Write(line.Value(source))
				}
			}
		case *ast.AutoLink:
			if entering {
				w.Write(n.URL(source))
			}
		case *ast.Link:
			if !entering {
				io.WriteString(w, " ("+string(n.Destination)+")")
			}
		case *discordmd.Inline:
			if n.Attr&discordmd.AttrSpoiler != 0 {
				if entering {
					io.WriteString(w, "*spoiler*")
				}
				return ast.WalkSkipChildren, nil
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
		case *discordmd.Mention:
			if entering {
				switch {
				case n.Channel != nil:
					io.WriteString(w, "#"+n.Channel.Name)
				case n.GuildUser != nil:
					io.WriteString(w, "@"+n.GuildUser.Username)
				case n.GuildRole != nil:
					io.WriteString(w, "@"+n.GuildRole.Name)
				}
			}
		case *discordmd.Emoji:
			if entering {
				io.WriteString(w, ":"+string(n.Name)+":")
			}
		}

		return ast.WalkContinue, nil
	})
}
