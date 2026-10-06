package messageslist

import (
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"log/slog"
	"net/http"
	"net/url"
	"slices"
	"strconv"

	"github.com/ayn2op/arikawa/v3/discord"
	"github.com/ayn2op/tview"
	tviewimage "github.com/ayn2op/tview/image"
	_ "golang.org/x/image/webp"
)

// Previews fit in a box of previewWidth by previewHeight cells, roughly square in pixels.
const (
	previewWidth  = 40
	previewHeight = 20
)

// kittyPixels is the width of a cell in pixels assumed for kitty previews when the terminal does not report it.
const kittyPixels = 10

type previewLoadedMsg struct {
	messageID discord.MessageID
	proxy     discord.URL
	preview   tviewimage.Widget
}

func (previewLoadedMsg) messagesList() {}

type previewSource struct {
	proxy         discord.URL
	width, height uint
}

func previewSources(message discord.Message) []previewSource {
	var sources []previewSource
	for _, a := range message.Attachments {
		sources = append(sources, previewSource{a.Proxy, a.Width, a.Height})
	}
	for _, e := range message.Embeds {
		switch {
		case e.Image != nil:
			sources = append(sources, previewSource{e.Image.Proxy, e.Image.Width, e.Image.Height})
		// Image and GIF embeds carry the media itself as the thumbnail.
		case e.Thumbnail != nil && (e.Type == discord.ImageEmbed || e.Type == discord.GIFVEmbed):
			sources = append(sources, previewSource{e.Thumbnail.Proxy, e.Thumbnail.Width, e.Thumbnail.Height})
		}
	}
	// Non-image files have no dimensions.
	return slices.DeleteFunc(sources, func(s previewSource) bool { return s.width == 0 || s.height == 0 })
}

// cols returns the preview's width in cells of cellWidth by cellHeight pixels: its own width at most, within the preview box.
func (s previewSource) cols(cellWidth, cellHeight uint) int {
	return int(max(min((s.width+cellWidth-1)/cellWidth, previewWidth, s.width*previewHeight*cellHeight/(s.height*cellWidth)), 1))
}

// previewCell returns the size of a cell in image pixels: the terminal's for kitty, and one pixel by two for half blocks.
func (m Model) previewCell() (width, height uint) {
	switch {
	case !m.kitty:
		return 1, 2
	case m.cellWidth == 0 || m.cellHeight == 0:
		return kittyPixels, kittyPixels * 2
	}
	return m.cellWidth, m.cellHeight
}

func (m Model) loadPreviews() tview.Cmd {
	if !m.cfg.Attachments.Preview {
		return nil
	}
	cellWidth, cellHeight := m.previewCell()
	var cmds []tview.Cmd
	for i := range m.items {
		item := &m.items[i]
		for _, s := range previewSources(item.message) {
			if _, ok := item.previews[s.proxy]; ok {
				continue
			}
			if item.previews == nil {
				item.previews = make(map[discord.URL]tviewimage.Widget)
			}
			cols := s.cols(cellWidth, cellHeight)
			width := min(s.width, uint(cols)*cellWidth)
			bounds := image.Rect(0, 0, int(width), int(s.height*width/s.width))
			widget := func(img image.Image) tviewimage.Widget {
				return tviewimage.New(img).Width(cols).CellSize(int(cellWidth), int(cellHeight))
			}
			// Reserve the preview's rows with a blank image of the same size, so the messages below do not shift when it loads.
			item.previews[s.proxy] = widget(image.NewAlpha(bounds))
			cmds = append(cmds, loadPreview(item.message.ID, s.proxy, bounds, widget))
		}
	}
	return tview.Batch(cmds...)
}

// loadPreview yields an empty preview if loading fails.
func loadPreview(messageID discord.MessageID, proxy discord.URL, bounds image.Rectangle, widget func(image.Image) tviewimage.Widget) tview.Cmd {
	return func() tview.Msg {
		img, err := downloadPreview(proxy, bounds)
		if err != nil {
			slog.Error("failed to load preview", "err", err)
			img = &image.Alpha{}
		}
		return previewLoadedMsg{messageID: messageID, proxy: proxy, preview: widget(img)}
	}
}

func downloadPreview(proxy discord.URL, bounds image.Rectangle) (image.Image, error) {
	u, err := url.Parse(proxy)
	if err != nil {
		return nil, err
	}
	query := u.Query()
	query.Set("width", strconv.Itoa(bounds.Dx()))
	query.Set("height", strconv.Itoa(bounds.Dy()))
	u.RawQuery = query.Encode()

	resp, err := http.Get(u.String())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	img, _, err := image.Decode(resp.Body)
	return img, err
}
