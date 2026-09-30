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

// kittyPixels is the number of pixels loaded per column of a kitty preview, roughly the width of a cell.
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

func (s previewSource) cols() int {
	// A cell is about twice as tall as wide.
	return int(max(min(s.width, previewWidth, s.width*previewHeight*2/s.height), 1))
}

func (ml *Model) loadPreviews() tview.Cmd {
	if !ml.cfg.Attachments.Preview {
		return nil
	}
	pixels := 1
	if ml.kitty {
		pixels = kittyPixels
	}
	var cmds []tview.Cmd
	for i := range ml.items {
		item := &ml.items[i]
		for _, s := range previewSources(item.message) {
			if _, ok := item.previews[s.proxy]; ok {
				continue
			}
			if item.previews == nil {
				item.previews = make(map[discord.URL]tviewimage.Widget)
			}
			width := min(s.width, uint(s.cols()*pixels))
			bounds := image.Rect(0, 0, int(width), int(s.height*width/s.width))
			// Reserve the preview's rows with a blank image of the same size, so the messages below do not shift when it loads.
			item.previews[s.proxy] = tviewimage.New(image.NewAlpha(bounds)).Width(s.cols())
			cmds = append(cmds, loadPreview(item.message.ID, s, bounds))
		}
	}
	return tview.Batch(cmds...)
}

// loadPreview yields an empty preview if loading fails.
func loadPreview(messageID discord.MessageID, s previewSource, bounds image.Rectangle) tview.Cmd {
	return func() tview.Msg {
		img, err := downloadPreview(s.proxy, bounds)
		if err != nil {
			slog.Error("failed to load preview", "err", err)
			img = &image.Alpha{}
		}
		return previewLoadedMsg{messageID: messageID, proxy: s.proxy, preview: tviewimage.New(img).Width(s.cols())}
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
