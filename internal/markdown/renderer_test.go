package markdown

import (
	"strings"
	"testing"

	md "github.com/ayn2op/arikawa/v3/markdown"
	"github.com/ayn2op/discordo/internal/config"
	"github.com/gdamore/tcell/v3"
)

func TestRendererRenderText(t *testing.T) {
	for _, test := range []struct {
		name   string
		source string
		mask   bool
		want   string
	}{
		{"spoilers_masked", "before ||secret|| after", true, "before [spoiler] after"},
		{"spoilers_visible", "before ||secret|| after", false, "before secret after"},
		{"ordered_list_after_nested", "1. a\n2. b\n   - x\n   - y\n3. c", false, "\n1. a\n2. b\n  - x\n  - y\n3. c"},
		{"ordered_list_after_nested_ordered", "1. a\n   1. x\n   2. y\n2. b", false, "\n1. a\n  1. x\n  2. y\n2. b"},
		{"list_nested_twice", "- a\n  - b\n    - c\n- d", false, "\n- a\n  - b\n    - c\n- d"},
		{"ordered_list_start", "3. a\n4. b", false, "\n3. a\n4. b"},
	} {
		t.Run(test.name, func(t *testing.T) {
			source := []byte(test.source)
			rendered := NewRenderer(&config.Config{Markdown: config.MarkdownConfig{MaskSpoilers: test.mask}}).RenderText(source, md.Parse(source), tcell.StyleDefault)
			lines := make([]string, len(rendered))
			for i, line := range rendered {
				for _, segment := range line {
					lines[i] += segment.Text
				}
			}
			if got := strings.Join(lines, "\n"); got != test.want {
				t.Fatalf("got %q, want %q", got, test.want)
			}
		})
	}
}
