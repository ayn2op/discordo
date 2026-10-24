package autocomplete

import (
	"image"
	"testing"

	"github.com/ayn2op/arikawa/v3/discord"
	"github.com/ayn2op/discordo/internal/ui"
	tviewimage "github.com/ayn2op/tview/image"
	"github.com/ayn2op/tview/text"
)

// item is an item drawn as its text.
type item string

func (i item) Insert() string { return string(i) }

func (i item) Line(map[discord.URL]tviewimage.Widget) []text.Span {
	return []text.Span{{Text: string(i)}}
}

func TestModelUpdate(t *testing.T) {
	t.Run("items", func(t *testing.T) {
		m := NewModel(nil)
		if _, ok := m.Selected(); ok || m.Visible() {
			t.Fatal("a new list shows items")
		}

		m, _ = m.Update(SetItemsMsg{"Emojis", []Item{item("😀"), item("😃")}})
		if item, ok := m.Selected(); !ok || item.Insert() != "😀" {
			t.Errorf("selected %v, %v, want the first item", item, ok)
		}

		if m, _ = m.Update(SetItemsMsg{}); m.Visible() {
			t.Error("a hidden list shows items")
		}
	})

	t.Run("images", func(t *testing.T) {
		cat := tviewimage.New(image.NewAlpha(image.Rect(0, 0, 2, 2)))
		m, _ := NewModel(nil).Update(SetItemsMsg{"Emojis", []Item{item("cat")}})
		if m, _ = m.Update(ui.ImagesUpdatedMsg{"cat": cat}); len(m.images) != 1 {
			t.Errorf("images = %v, want the image", m.images)
		}

		if m, _ = m.Update(SetItemsMsg{"Emojis", []Item{item("dog")}}); len(m.images) != 0 {
			t.Errorf("images = %v after showing other items, want none", m.images)
		}
	})
}
