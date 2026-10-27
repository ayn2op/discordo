package chat

import (
	"compress/zlib"
	"image"
	"image/draw"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"log/slog"
	"maps"
	"net/url"
	"slices"
	"strconv"
	"sync"

	"github.com/ayn2op/arikawa/v3/discord"
	"github.com/ayn2op/discordo/internal/ui/images"
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

// compressorPool reuses compressors, as each holds about a megabyte of state.
var compressorPool = sync.Pool{New: func() any {
	compressor, _ := zlib.NewWriterLevel(nil, zlib.BestSpeed)
	return compressor
}}

// pooledCompressor returns its writer to the pool on Close.
type pooledCompressor struct {
	*zlib.Writer
}

func newPooledCompressor(w io.Writer) (io.WriteCloser, error) {
	compressor := compressorPool.Get().(*zlib.Writer)
	compressor.Reset(w)
	return pooledCompressor{compressor}, nil
}

func (compressor pooledCompressor) Close() error {
	err := compressor.Writer.Close()
	compressorPool.Put(compressor.Writer)
	return err
}

// WezTerm and Konsole support kitty graphics but not Unicode placeholders.
var placeholderTerminals = []string{"kitty", "ghostty", "Rio"}

// imageCell returns the size of a cell in image pixels.
func (m Model) imageCell() image.Point {
	switch {
	case !m.kitty:
		return image.Pt(1, 2)
	case m.cellWidth == 0 || m.cellHeight == 0:
		return image.Pt(kittyPixels, kittyPixels*2)
	}
	return image.Pt(int(m.cellWidth), int(m.cellHeight))
}

// reloadImages loads the shown images again if the cell size is no longer cell, as they were sized for it.
func (m *Model) reloadImages(cell image.Point) tview.Cmd {
	if m.imageCell() == cell || len(m.images) == 0 {
		return nil
	}

	return tview.Batch(m.clearImages(), m.messagesList.LoadImages())
}

// clearImages drops the images and returns a command that frees them in the terminal.
func (m *Model) clearImages() tview.Cmd {
	var cmds []tview.Cmd
	for _, image := range m.images {
		cmds = append(cmds, image.Delete())
	}

	clear(m.images)
	return tview.Batch(append(cmds, m.sendImages())...)
}

// sendImages passes a copy of the images to the models that draw them.
func (m *Model) sendImages() tview.Cmd {
	images := images.ChangedMsg(maps.Clone(m.images))
	var cmd tview.Cmd
	m.reactionsPicker, cmd = m.reactionsPicker.Update(images)
	return tview.Batch(cmd, m.updatePane(paneMessagesList, images), m.updatePane(paneComposer, images))
}

func (m *Model) loadImages(msg images.RequestedMsg) tview.Cmd {
	cell := m.imageCell()
	cellWidth, cellHeight := uint(cell.X), uint(cell.Y)
	var cmds []tview.Cmd
	for url, rows := range msg.Squares {
		cols := squareCols(rows, cell)
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
	cell := m.imageCell()
	return func() tview.Msg {
		img, err := m.downloadImage(url, bounds)
		if err != nil {
			slog.Error("failed to load image", "url", url, "err", err)
			img = &image.Alpha{}
		}
		return images.LoadedMsg{URL: url, Image: widget(img), Cell: cell}
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

// squareCols returns the width in cells of a square image rows tall.
func squareCols(rows uint, cell image.Point) uint {
	return rows * uint(cell.Y) / uint(cell.X)
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
