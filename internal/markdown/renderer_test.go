package markdown

import (
	"strings"
	"testing"

	md "github.com/ayn2op/arikawa/v3/markdown"
	"github.com/ayn2op/discordo/internal/config"
	"github.com/gdamore/tcell/v3"
)

func TestRendererRenderText(t *testing.T) {
	source := []byte("before ||secret|| after")
	node := md.Parse(source)

	for _, test := range []struct {
		name string
		mask bool
		want string
	}{
		{"spoilers_masked", true, "before [spoiler] after"},
		{"spoilers_visible", false, "before secret after"},
	} {
		t.Run(test.name, func(t *testing.T) {
			rendered := NewRenderer(&config.Config{Markdown: config.MarkdownConfig{MaskSpoilers: test.mask}}).RenderText(source, node, tcell.StyleDefault)
			var got strings.Builder
			for _, line := range rendered {
				for _, segment := range line {
					got.WriteString(segment.Text)
				}
			}
			if got.String() != test.want {
				t.Fatalf("mask=%t: got %q, want %q", test.mask, got.String(), test.want)
			}
		})
	}
}
