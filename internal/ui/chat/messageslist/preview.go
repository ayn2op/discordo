package messageslist

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
	md "github.com/ayn2op/arikawa/v3/markdown"
	"github.com/ayn2op/tview"
	tviewimage "github.com/ayn2op/tview/image"
	"github.com/ayn2op/tview/text"
	"github.com/yuin/goldmark/v2/ast"
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

func (m Model) previewSources(message discord.Message) []previewSource {
	var sources []previewSource
	for _, attachment := range message.Attachments {
		sources = append(sources, previewSource{attachment.Proxy, attachment.Width, attachment.Height})
	}

	embeds := message.Embeds
	if !m.cfg.UI.MessagesList.Embeds.Enabled {
		embeds = nil
	}
	for _, embed := range embeds {
		switch {
		case embed.Image != nil:
			sources = append(sources, previewSource{embed.Image.Proxy, embed.Image.Width, embed.Image.Height})
		// Image and GIF embeds carry the media itself as the thumbnail.
		case embed.Thumbnail != nil && (embed.Type == discord.ImageEmbed || embed.Type == discord.GIFVEmbed):
			sources = append(sources, previewSource{embed.Thumbnail.Proxy, embed.Thumbnail.Width, embed.Thumbnail.Height})
		}
	}

	// Non-image files have no dimensions.
	return slices.DeleteFunc(sources, func(source previewSource) bool { return source.width == 0 || source.height == 0 })
}

// cols returns the preview's width in cells of cellWidth by cellHeight pixels: its own width at most, within the preview box.
func (s previewSource) cols(cellWidth, cellHeight uint) int {
	// The columns its own width takes, rounded up, so it is not scaled up.
	own := (s.width + cellWidth - 1) / cellWidth
	// The columns at which its height fills the box, so a tall image is not cut off.
	tall := s.width * previewHeight * cellHeight / (s.height * cellWidth)
	return int(max(min(own, previewWidth, tall), 1))
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
	cellWidth, cellHeight := m.previewCell()
	var cmds []tview.Cmd
	for index := range m.items {
		item := &m.items[index]
		if item.separator != nil {
			continue
		}

		cmds = append(cmds, m.loadAvatar(item.message.Author))
		cmds = append(cmds, m.loadEmojis(item)...)
		if m.cfg.UI.MessagesList.Attachments.Preview {
			cmds = append(cmds, m.loadMessagePreviews(item, cellWidth, cellHeight)...)
		}
	}

	return tview.Batch(cmds...)
}

// loadEmojis loads the custom emoji of the reactions to the item's message and in its content.
func (m Model) loadEmojis(item *messageItem) []tview.Cmd {
	var cmds []tview.Cmd
	for _, reaction := range item.message.Reactions {
		if reaction.Emoji.IsCustom() {
			cmds = append(cmds, m.loadEmoji(reaction.Emoji.ID))
		}
	}

	root := m.contentRoot(item)
	if root == nil {
		return cmds
	}

	_ = ast.Walk(root, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		emoji, ok := node.(*md.Emoji)
		if !ok || !entering {
			return ast.WalkContinue, nil
		}

		if id, err := discord.ParseSnowflake(emoji.ID); err == nil {
			cmds = append(cmds, m.loadEmoji(discord.EmojiID(id)))
		}

		return ast.WalkContinue, nil
	})

	return cmds
}

// loadMessagePreviews loads the previews of the item's message that are not loaded yet.
func (m Model) loadMessagePreviews(item *messageItem, cellWidth, cellHeight uint) []tview.Cmd {
	var cmds []tview.Cmd
	for _, source := range m.previewSources(item.message) {
		if _, ok := item.previews[source.proxy]; ok {
			continue
		}

		if item.previews == nil {
			item.previews = make(map[discord.URL]tviewimage.Widget)
		}

		cols := source.cols(cellWidth, cellHeight)
		width := min(source.width, uint(cols)*cellWidth)
		bounds := image.Rect(0, 0, int(width), int(source.height*width/source.width))
		widget := func(img image.Image) tviewimage.Widget {
			return tviewimage.New(img).Width(cols).CellSize(int(cellWidth), int(cellHeight))
		}

		// Reserve the preview's rows with a blank image of the same size, so the messages below do not shift when it loads.
		item.previews[source.proxy] = widget(image.NewAlpha(bounds))
		cmds = append(cmds, m.loadPreview(item.message.ID, source.proxy, bounds, widget))
	}

	return cmds
}

func avatarURL(user discord.User) discord.URL {
	return user.AvatarURLWithType(discord.PNGImage) + "?size=128"
}

func (m Model) loadAvatar(user discord.User) tview.Cmd {
	cfg := m.cfg.UI.MessagesList.Avatars
	if !cfg.Preview || cfg.Height == 0 || m.blocked(user) {
		return nil
	}
	return m.loadImage(avatarURL(user), uint(cfg.Height))
}

func emojiURL(id discord.EmojiID) discord.URL {
	return discord.Emoji{ID: id}.EmojiURLWithType(discord.PNGImage)
}

// loadEmoji loads the image of the custom emoji with id, a row tall.
func (m Model) loadEmoji(id discord.EmojiID) tview.Cmd {
	return m.loadImage(emojiURL(id), 1)
}

// emojiLine returns the image of the custom emoji with id as a line of spans, or nil if it is not loaded.
func (m Model) emojiLine(id string) []text.Span {
	snowflake, err := discord.ParseSnowflake(id)
	if err != nil {
		return nil
	}

	emoji, ok := m.images[emojiURL(discord.EmojiID(snowflake))]
	if !ok {
		return nil
	}

	if lines := emoji.Lines(); len(lines) > 0 {
		return lines[0]
	}
	return nil
}

// loadImage loads the image at url in a square rows tall, unless it is loaded already.
func (m Model) loadImage(url discord.URL, rows uint) tview.Cmd {
	if _, ok := m.images[url]; ok {
		return nil
	}

	cellWidth, cellHeight := m.previewCell()
	cols := rows * cellHeight / cellWidth
	bounds := image.Rect(0, 0, int(cols*cellWidth), int(cols*cellWidth))
	widget := func(img image.Image) tviewimage.Widget {
		return tviewimage.New(square(img)).Width(int(cols)).CellSize(int(cellWidth), int(cellHeight))
	}
	m.images[url] = widget(image.NewAlpha(bounds))
	return m.loadPreview(0, url, bounds, widget)
}

// square returns img centered on a transparent square, so an image of any shape takes the square's cells.
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

// loadPreview yields an empty preview if loading fails.
func (m Model) loadPreview(messageID discord.MessageID, proxy discord.URL, bounds image.Rectangle, widget func(image.Image) tviewimage.Widget) tview.Cmd {
	return func() tview.Msg {
		img, err := m.downloadPreview(proxy, bounds)
		if err != nil {
			slog.Error("failed to load preview", "err", err)
			img = &image.Alpha{}
		}
		return previewLoadedMsg{messageID: messageID, proxy: proxy, preview: widget(img)}
	}
}

func (m Model) downloadPreview(proxy discord.URL, bounds image.Rectangle) (image.Image, error) {
	previewURL, err := url.Parse(proxy)
	if err != nil {
		return nil, err
	}

	query := previewURL.Query()
	query.Set("width", strconv.Itoa(bounds.Dx()))
	query.Set("height", strconv.Itoa(bounds.Dy()))
	previewURL.RawQuery = query.Encode()

	resp, err := m.get(previewURL.String())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close() //nolint:errcheck
	img, _, err := image.Decode(resp.Body)
	return img, err
}
