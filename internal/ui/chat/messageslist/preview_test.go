package messageslist

import (
	"image"
	"slices"
	"testing"

	"github.com/ayn2op/arikawa/v3/discord"
	tviewimage "github.com/ayn2op/tview/image"
)

func TestModelPreviewSources(t *testing.T) {
	for _, tt := range []struct {
		name    string
		message discord.Message
		want    []previewSource
	}{
		{"file attachment", discord.Message{Attachments: []discord.Attachment{{Proxy: "a"}}}, nil},
		{"embed image over thumbnail", discord.Message{Embeds: []discord.Embed{{
			Type:      discord.ImageEmbed,
			Image:     &discord.EmbedImage{Proxy: "i", Width: 2, Height: 1},
			Thumbnail: &discord.EmbedThumbnail{Proxy: "t", Width: 2, Height: 1},
		}}}, []previewSource{{"i", 2, 1}}},
		{"gif thumbnail", discord.Message{Embeds: []discord.Embed{{
			Type:      discord.GIFVEmbed,
			Thumbnail: &discord.EmbedThumbnail{Proxy: "t", Width: 2, Height: 1},
		}}}, []previewSource{{"t", 2, 1}}},
		{"article thumbnail", discord.Message{Embeds: []discord.Embed{{
			Type:      discord.ArticleEmbed,
			Thumbnail: &discord.EmbedThumbnail{Proxy: "t", Width: 2, Height: 1},
		}}}, nil},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ml := newTestModel(t, testConfig(t))
			if got := ml.previewSources(tt.message); !slices.Equal(got, tt.want) {
				t.Fatalf("previewSources() = %v, want %v", got, tt.want)
			}
		})
	}

	t.Run("embeds disabled", func(t *testing.T) {
		ml := newTestModel(t, testConfig(t))
		ml.cfg.UI.MessagesList.Embeds.Enabled = false
		message := discord.Message{
			Attachments: []discord.Attachment{{Proxy: "a", Width: 2, Height: 1}},
			Embeds: []discord.Embed{{
				Type:  discord.ImageEmbed,
				Image: &discord.EmbedImage{Proxy: "i", Width: 2, Height: 1},
			}},
		}
		want := []previewSource{{"a", 2, 1}}
		if got := ml.previewSources(message); !slices.Equal(got, want) {
			t.Fatalf("previewSources() = %v, want %v", got, want)
		}
	})
}

func TestPreviewSourceCols(t *testing.T) {
	for _, tt := range []struct {
		name                  string
		width, height         uint
		cellWidth, cellHeight uint
		want                  int
	}{
		{"landscape", 1600, 900, 1, 2, 40},
		{"portrait", 900, 1600, 1, 2, 22},
		{"small", 10, 10, 1, 2, 10},
		{"very tall", 1, 10000, 1, 2, 1},
		{"own width in cells", 200, 100, 10, 20, 20},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := (previewSource{width: tt.width, height: tt.height}).cols(tt.cellWidth, tt.cellHeight); got != tt.want {
				t.Fatalf("cols() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestSquare(t *testing.T) {
	t.Run("square", func(t *testing.T) {
		img := image.NewAlpha(image.Rect(0, 0, 20, 20))
		if got := square(img); got != image.Image(img) {
			t.Fatal("square image was copied")
		}
	})
	t.Run("tall", func(t *testing.T) {
		img := image.NewAlpha(image.Rect(0, 0, 10, 30))
		img.Pix[0] = 0xff
		got := square(img)
		if got.Bounds() != image.Rect(0, 0, 30, 30) {
			t.Fatalf("bounds = %v, want 30 by 30", got.Bounds())
		}
		if _, _, _, a := got.At(10, 0).RGBA(); a == 0 {
			t.Fatal("image is not centered")
		}
	})
}

func TestModelEmojiLine(t *testing.T) {
	ml := newTestModel(t, testConfig(t))
	if got := ml.emojiLine("5"); got != nil {
		t.Fatalf("line = %v, want nil before the emoji is loaded", got)
	}

	ml.images[emojiURL(5)] = tviewimage.New(image.NewAlpha(image.Rect(0, 0, 2, 2))).Width(2)
	if got := ml.emojiLine("5"); len(got) != 2 {
		t.Fatalf("line = %v, want the two cells of the emoji", got)
	}
}

func TestModelLoadPreviews(t *testing.T) {
	ml := newTestModel(t, testConfig(t))
	ml.setMessages([]discord.Message{{ID: 2, ChannelID: 1, Content: "hi <:cat:5>"}})
	ml.loadPreviews()
	if _, ok := ml.images[emojiURL(5)]; !ok {
		t.Fatal("the emoji in the content was not loaded")
	}

	item := &ml.items[len(ml.items)-1]
	if item.root == nil {
		t.Fatal("the parsed content was not kept for rendering")
	}
}
