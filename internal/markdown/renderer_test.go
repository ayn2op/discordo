package markdown

import (
	"strings"
	"testing"

	md "github.com/ayn2op/arikawa/v3/markdown"
	"github.com/ayn2op/discordo/internal/config"
	"github.com/gdamore/tcell/v3"
)

func TestRendererRenderText(t *testing.T) {
	t.Run("masks spoilers", func(t *testing.T) {
		source := []byte("before ||secret|| after")
		node := md.Parse(source)

		for _, test := range []struct {
			mask bool
			want string
		}{
			{true, "before [spoiler] after"},
			{false, "before secret after"},
		} {
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
		}
	})
}
