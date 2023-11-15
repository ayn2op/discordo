package md

import (
	"io"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/renderer"
)

var (
	resetTag = []byte("[::-]")

	boldTag    = []byte("[::b]")
	italicTag  = []byte("[::i]")
	reverseTag = []byte("[::r]")
)

type Renderer struct{}

func newRenderer() *Renderer {
	return &Renderer{}
}

func (r *Renderer) Render(w io.Writer, source []byte, n ast.Node) error {
	return ast.Walk(n, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		switch n := n.(type) {
		case *ast.Document:
			// no-op

		case *ast.FencedCodeBlock:
			if entering {
				io.WriteString(w, "[::r]")
			} else {
				w.Write(resetTag)
			}
		case *ast.CodeBlock:
			ls := n.Lines()
			for i := 0; i < ls.Len(); i++ {
				l := ls.At(i)
				io.WriteString(w, string(l.Value(source)))
			}

		case *ast.Emphasis:
			var tag string
			switch n.Level {
			case 1: // italic
				tag = "[::i]"
			case 2: // bold
				tag = "[::b]"
			}

			if entering {
				io.WriteString(w, tag)
			} else {
				w.Write(resetTag)
			}

		case *ast.List:
			if entering {
				io.WriteString(w, "\n")
			}
		case *ast.ListItem:
			if entering {
				io.WriteString(w, "- ")
			}

		case *ast.Text:
			if entering {
				w.Write(n.Segment.Value(source))
			}
		}

		return ast.WalkContinue, nil
	})
}

// AddOptions adds given option to this renderer.
func (r *Renderer) AddOptions(...renderer.Option) {}
