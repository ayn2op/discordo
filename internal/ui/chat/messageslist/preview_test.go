package messageslist

import (
	"slices"
	"testing"

	"github.com/ayn2op/arikawa/v3/discord"
)

func TestPreviewSources(t *testing.T) {
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
			if got := previewSources(tt.message); !slices.Equal(got, tt.want) {
				t.Fatalf("previewSources() = %v, want %v", got, tt.want)
			}
		})
	}
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
