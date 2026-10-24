package ui

import (
	"image"

	"github.com/ayn2op/arikawa/v3/discord"
	tviewimage "github.com/ayn2op/tview/image"
)

// EmojiRows is how tall custom emojis are drawn, the same in every pane, as they share the images by URL.
const EmojiRows = 1

// LoadImagesMsg asks for images: Squares by rows tall, Previews by size in pixels.
type LoadImagesMsg struct {
	Squares  map[discord.URL]uint
	Previews map[discord.URL]image.Point
}

// ImageLoadedMsg is an image that finished loading, or an empty one if it failed.
type ImageLoadedMsg struct {
	URL   discord.URL
	Image tviewimage.Widget
	// Cell is the size of a cell in image pixels that the image was sized for.
	Cell image.Point
}

// ImagesUpdatedMsg holds images by URL, blank while they load.
type ImagesUpdatedMsg map[discord.URL]tviewimage.Widget
