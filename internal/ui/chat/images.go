package chat

import (
	"image"
	"image/draw"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"log/slog"
	"net/url"
	"slices"
	"strconv"

	"github.com/ayn2op/arikawa/v3/discord"
	"github.com/ayn2op/discordo/internal/ui"
	"github.com/ayn2op/tview"
	tviewimage "github.com/ayn2op/tview/image"
	_ "golang.org/x/image/webp"
)

// Roughly square in pixels.
const (
	previewWidth  = 40
	previewHeight = 20
)

// kittyPixels is the assumed cell width when the terminal does not report it.
const kittyPixels = 10

// maxKittyImages is how many images kitty's placeholders tell apart.
const maxKittyImages = 63488

// WezTerm and Konsole support kitty graphics but not Unicode placeholders.
var placeholderTerminals = []string{"kitty", "ghostty", "Rio"}

// imageCell returns the size of a cell in image pixels.
func (m Model) imageCell() (width, height uint) {
	switch {
	case !m.kitty:
		return 1, 2
	case m.cellWidth == 0 || m.cellHeight == 0:
		return kittyPixels, kittyPixels * 2
	}
	return m.cellWidth, m.cellHeight
}

func (m *Model) loadImages(msg ui.LoadImagesMsg) tview.Cmd {
	cellWidth, cellHeight := m.imageCell()
	var cmds []tview.Cmd
	for url, rows := range msg.Squares {
		cols := rows * cellHeight / cellWidth
		bounds := image.Rect(0, 0, int(cols*cellWidth), int(cols*cellWidth))
		cmds = append(cmds, m.loadImage(url, bounds, func(img image.Image) tviewimage.Widget {
			return tviewimage.New(square(img)).Width(int(cols)).CellSize(int(cellWidth), int(cellHeight))
		}))
	}

	for url, size := range msg.Previews {
		cols := previewCols(uint(size.X), uint(size.Y), cellWidth, cellHeight)
		width := min(uint(size.X), uint(cols)*cellWidth)
		bounds := image.Rect(0, 0, int(width), int(uint(size.Y)*width/uint(size.X)))
		cmds = append(cmds, m.loadImage(url, bounds, func(img image.Image) tviewimage.Widget {
			return tviewimage.New(img).Width(cols).CellSize(int(cellWidth), int(cellHeight))
		}))
	}

	return tview.Batch(cmds...)
}

// A blank image holds the place until it loads, so nothing shifts.
func (m *Model) loadImage(url discord.URL, bounds image.Rectangle, widget func(image.Image) tviewimage.Widget) tview.Cmd {
	if _, ok := m.images[url]; ok {
		return nil
	}

	m.images[url] = widget(image.NewAlpha(bounds))
	return func() tview.Msg {
		img, err := m.downloadImage(url, bounds)
		if err != nil {
			slog.Error("failed to load image", "url", url, "err", err)
			img = &image.Alpha{}
		}
		return ui.ImageLoadedMsg{URL: url, Image: widget(img)}
	}
}

// downloadImage expects proxy to be a Discord media proxy URL, which scales the image to bounds.
func (m Model) downloadImage(proxy discord.URL, bounds image.Rectangle) (image.Image, error) {
	previewURL, err := url.Parse(proxy)
	if err != nil {
		return nil, err
	}

	query := previewURL.Query()
	query.Set("width", strconv.Itoa(bounds.Dx()))
	query.Set("height", strconv.Itoa(bounds.Dy()))
	previewURL.RawQuery = query.Encode()

	resp, err := m.messagesList.Get(previewURL.String())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close() //nolint:errcheck
	img, _, err := image.Decode(resp.Body)
	return img, err
}

// isKitty reports whether images are drawn with kitty's placeholders.
func (m Model) isKitty(name string) bool {
	switch m.cfg.UI.MessagesList.ImageProtocol {
	case "kitty":
		return true
	case "halfblocks":
		return false
	}
	return slices.Contains(placeholderTerminals, name)
}

// previewCols returns the width of a preview in cells.
func previewCols(width, height, cellWidth, cellHeight uint) int {
	// Not scaled up.
	own := (width + cellWidth - 1) / cellWidth
	// A tall image is not cut off.
	tall := width * previewHeight * cellHeight / (height * cellWidth)
	return int(max(min(own, previewWidth, tall), 1))
}

// square centers img on a transparent square.
func square(img image.Image) image.Image {
	bounds := img.Bounds()
	if bounds.Dx() == bounds.Dy() {
		return img
	}

	side := max(bounds.Dx(), bounds.Dy())
	dst := image.NewNRGBA(image.Rect(0, 0, side, side))
	at := image.Pt((side-bounds.Dx())/2, (side-bounds.Dy())/2)
	draw.Draw(dst, bounds.Sub(bounds.Min).Add(at), img, bounds.Min, draw.Src)
	return dst
}
