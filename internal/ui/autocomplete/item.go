package autocomplete

import (
	"github.com/ayn2op/arikawa/v3/discord"
	tviewimage "github.com/ayn2op/tview/image"
	"github.com/ayn2op/tview/text"
)

// Item is a suggestion.
type Item interface {
	// Insert returns the text that replaces the word being completed.
	Insert() string
	// Line returns the item as drawn, with images by URL, which may not be loaded yet.
	Line(images map[discord.URL]tviewimage.Widget) []text.Span
}
