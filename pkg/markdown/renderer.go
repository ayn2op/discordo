package markdown

import (
	"io"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/renderer"
)

type discordRenderer struct{}

func (r *discordRenderer) Render(w io.Writer, source []byte, root ast.Node) error {
	ast.Walk(root, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		switch n := n.(type) {
		case *ast.Document:
		// NOOP
		case *ast.String:
			if entering {
				w.Write(n.Value)
			}
		case *ast.Text:
			if entering {
				w.Write(n.Segment.Value(source))

				switch {
				case n.SoftLineBreak():
					io.WriteString(w, "\n")
				case n.HardLineBreak():
					io.WriteString(w, "\n\n")
				}
			}
		case *ast.Emphasis:
			var tag string
			switch n.Level {
			case 1:
				tag = "[::i]"
			case 2:
				tag = "[::b]"
			}

			if entering {
				io.WriteString(w, tag)
			} else {
				io.WriteString(w, "[::-]")
			}
		case *ast.CodeSpan:
			if entering {
				io.WriteString(w, "[::r]")
			} else {
				io.WriteString(w, "[::-]")
			}
		}

		return ast.WalkContinue, nil
	})

	return nil
}

func (r *discordRenderer) AddOptions(...renderer.Option) {}
