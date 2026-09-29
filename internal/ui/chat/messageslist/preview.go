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

// previewWidth is the widest a preview is drawn, in cells.
const previewWidth = 40

type previewLoadedMsg struct {
	messageID discord.MessageID
	proxy     discord.URL
	image     image.Image
}

func (previewLoadedMsg) messagesList() {}

// previewSource is an image to preview.
type previewSource struct {
	proxy         discord.URL
	width, height uint
}

// previewSources returns the images of message's attachments and embeds.
func previewSources(message discord.Message) []previewSource {
	var sources []previewSource
	for _, a := range message.Attachments {
		sources = append(sources, previewSource{a.Proxy, a.Width, a.Height})
	}
	for _, e := range message.Embeds {
		switch {
		case e.Image != nil:
			sources = append(sources, previewSource{e.Image.Proxy, e.Image.Width, e.Image.Height})
		// Image links and GIFs have their media as the thumbnail.
		case e.Thumbnail != nil && (e.Type == discord.ImageEmbed || e.Type == discord.GIFVEmbed):
			sources = append(sources, previewSource{e.Thumbnail.Proxy, e.Thumbnail.Width, e.Thumbnail.Height})
		}
	}
	// Files that are not images have no size.
	return slices.DeleteFunc(sources, func(s previewSource) bool { return s.width == 0 || s.height == 0 })
}

// bounds returns the size s is loaded at, a pixel per cell of its preview.
func (s previewSource) bounds() image.Rectangle {
	width := min(s.width, previewWidth)
	return image.Rect(0, 0, int(width), int(s.height*width/s.width))
}

// loadPreviews loads the previews of the shown messages that are not loaded yet.
func (ml *Model) loadPreviews() tview.Cmd {
	if !ml.cfg.Attachments.Preview {
		return nil
	}
	var cmds []tview.Cmd
	for i := range ml.items {
		item := &ml.items[i]
		for _, s := range previewSources(item.message) {
			if _, ok := item.previews[s.proxy]; ok {
				continue
			}
			if item.previews == nil {
				item.previews = make(map[discord.URL]image.Image)
			}
			// A transparent image of the preview's size reserves its rows while it loads, so the messages below do not jump.
			item.previews[s.proxy] = image.NewAlpha(s.bounds())
			cmds = append(cmds, loadPreview(item.message.ID, s))
		}
	}
	return tview.Batch(cmds...)
}

// loadPreview loads the preview of s, with a nil image if it fails to.
func loadPreview(messageID discord.MessageID, s previewSource) tview.Cmd {
	return func() tview.Msg {
		img, err := downloadPreview(s)
		if err != nil {
			slog.Error("failed to load preview", "err", err)
		}
		return previewLoadedMsg{messageID: messageID, proxy: s.proxy, image: img}
	}
}

// downloadPreview downloads s from the media proxy, scaled down to its bounds.
func downloadPreview(s previewSource) (image.Image, error) {
	u, err := url.Parse(s.proxy)
	if err != nil {
		return nil, err
	}
	bounds := s.bounds()
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

// preview returns the widget that draws img within width cells.
func preview(img image.Image, width int) tviewimage.Widget {
	return tviewimage.New(img).Width(min(width, previewWidth, img.Bounds().Dx()))
}
