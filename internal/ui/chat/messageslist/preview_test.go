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
