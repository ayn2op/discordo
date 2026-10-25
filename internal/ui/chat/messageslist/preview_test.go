package messageslist

import (
	"image"
	"slices"
	"testing"

	"github.com/ayn2op/arikawa/v3/discord"
	"github.com/ayn2op/discordo/internal/ui"
	"github.com/ayn2op/discordo/internal/ui/images"
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

func TestModelEmojiLine(t *testing.T) {
	ml := newTestModel(t, testConfig(t))
	if got := ml.emojiLine("5"); got != nil {
		t.Fatalf("line = %v, want nil before the emoji is loaded", got)
	}

	ml.images = map[discord.URL]tviewimage.Widget{emojiURL(5): tviewimage.New(image.NewAlpha(image.Rect(0, 0, 2, 2))).Width(2)}
	if got := ml.emojiLine("5"); len(got) != 2 {
		t.Fatalf("line = %v, want the two cells of the emoji", got)
	}
}

func TestModelLoadImages(t *testing.T) {
	ml := newTestModel(t, testConfig(t))
	ml.setMessages([]discord.Message{{ID: 2, ChannelID: 1, Content: "hi <:cat:5>"}})
	msg := ml.LoadImages()().(images.RequestedMsg)
	if rows := msg.Squares[emojiURL(5)]; rows != ui.EmojiRows {
		t.Fatalf("the emoji in the content was asked for %d rows tall, want %d", rows, ui.EmojiRows)
	}

	item := &ml.items[len(ml.items)-1]
	if item.root == nil {
		t.Fatal("the parsed content was not kept for rendering")
	}
}
