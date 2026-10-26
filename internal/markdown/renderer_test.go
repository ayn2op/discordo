package markdown

import (
	"strings"
	"testing"

	md "github.com/ayn2op/arikawa/v3/markdown"
	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/tview/text"
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
			rendered := NewRenderer(&config.Config{Markdown: config.MarkdownConfig{MaskSpoilers: test.mask}}).RenderText(source, md.Parse(source), tcell.StyleDefault, func(string) []text.Span { return nil })
			var got strings.Builder
			for _, span := range rendered {
				got.WriteString(span.Text)
			}
			if got.String() != test.want {
				t.Fatalf("got %q, want %q", got.String(), test.want)
			}
		})
	}

	t.Run("emoji", func(t *testing.T) {
		source := []byte("hi <:cat:5> <:dog:6>")
		// Only the emoji with ID 5 has an image.
		emoji := func(id string) []text.Span {
			if id == "5" {
				return []text.Span{{Text: "I"}}
			}
			return nil
		}

		for _, test := range []struct {
			name     string
			colons   bool
			showName bool
			want     string
		}{
			{"plain", false, true, "hi I cat dog"},
			{"colons", true, true, "hi I :cat: :dog:"},
			// The emoji without an image keeps its name.
			{"image only", false, false, "hi I dog"},
		} {
			t.Run(test.name, func(t *testing.T) {
				var cfg config.Config
				cfg.UI.MessagesList.Emojis.ColonsAroundName = test.colons
				cfg.UI.MessagesList.Emojis.ShowEmojiName = test.showName
				var got strings.Builder
				for _, span := range NewRenderer(&cfg).RenderText(source, md.Parse(source), tcell.StyleDefault, emoji) {
					got.WriteString(span.Text)
				}
				if got.String() != test.want {
					t.Fatalf("got %q, want %q", got.String(), test.want)
				}
			})
		}
	})
}
