package config

import (
	"github.com/rivo/tview"
)

type BorderPreset struct {
	Horizontal  rune
	Vertical    rune
	TopLeft     rune
	TopRight    rune
	BottomLeft  rune
	BottomRight rune
}

func (p *BorderPreset) UnmarshalTOML(v any) error {
	switch v.(string) {
	case "double":
		*p = BorderPreset{
			Horizontal:  tview.BoxDrawingsDoubleHorizontal,
			Vertical:    tview.BoxDrawingsDoubleVertical,
			TopLeft:     tview.BoxDrawingsDoubleDownAndRight,
			TopRight:    tview.BoxDrawingsDoubleDownAndLeft,
			BottomLeft:  tview.BoxDrawingsDoubleUpAndRight,
			BottomRight: tview.BoxDrawingsDoubleUpAndLeft,
		}
	case "thick":
		*p = BorderPreset{
			Horizontal:  tview.BoxDrawingsHeavyHorizontal,
			Vertical:    tview.BoxDrawingsHeavyVertical,
			TopLeft:     tview.BoxDrawingsHeavyDownAndRight,
			TopRight:    tview.BoxDrawingsHeavyDownAndLeft,
			BottomLeft:  tview.BoxDrawingsHeavyUpAndRight,
			BottomRight: tview.BoxDrawingsHeavyUpAndLeft,
		}
	case "round":
		*p = BorderPreset{
			Horizontal:  tview.BoxDrawingsLightHorizontal,
			Vertical:    tview.BoxDrawingsLightVertical,
			TopLeft:     tview.BoxDrawingsLightArcDownAndRight,
			TopRight:    tview.BoxDrawingsLightArcDownAndLeft,
			BottomLeft:  tview.BoxDrawingsLightArcUpAndRight,
			BottomRight: tview.BoxDrawingsLightArcUpAndLeft,
		}
	case "hidden":
		*p = BorderPreset{
			Horizontal:  ' ',
			Vertical:    ' ',
			TopLeft:     ' ',
			TopRight:    ' ',
			BottomLeft:  ' ',
			BottomRight: ' ',
		}
	}

	return nil
}

type (
	BorderTheme struct {
		Enabled bool   `toml:"enabled"`
		Padding [4]int `toml:"padding"`

		Color       string       `toml:"color"`
		ActiveColor string       `toml:"active_color"`
		Preset      BorderPreset `toml:"preset"`
	}

	Theme struct {
		TitleColor      string            `toml:"title_color"`
		BackgroundColor string            `toml:"background_color"`
		Border          BorderTheme       `toml:"border"`
		GuildsTree      GuildsTreeTheme   `toml:"guilds_tree"`
		MessagesText    MessagesTextTheme `toml:"messages_text"`
	}

	GuildsTreeTheme struct {
		AutoExpandFolders bool `toml:"auto_expand_folders"`
		Graphics          bool `toml:"graphics"`

		PrivateChannelColor string `toml:"private_channel_color"`
		GuildColor          string `toml:"guild_color"`
		ChannelColor        string `toml:"channel_color"`
	}

	MessagesTextTheme struct {
		ReplyIndicator string `toml:"reply_indicator"`

		AuthorColor     string `toml:"author_color"`
		ContentColor    string `toml:"content_color"`
		EmojiColor      string `toml:"emoji_color"`
		LinkColor       string `toml:"link_color"`
		AttachmentColor string `toml:"attachment_color"`
	}
)

func defaultTheme() Theme {
	return Theme{
		Border: BorderTheme{
			Enabled: true,
			Padding: [...]int{0, 0, 1, 1},

			Color:       "default",
			ActiveColor: "green",
			Preset: BorderPreset{
				Horizontal:  tview.Borders.Horizontal,
				Vertical:    tview.Borders.Vertical,
				TopLeft:     tview.Borders.TopLeft,
				TopRight:    tview.Borders.TopRight,
				BottomLeft:  tview.Borders.BottomLeft,
				BottomRight: tview.Borders.BottomRight,
			},
		},

		BackgroundColor: "default",
		TitleColor:      "default",

		GuildsTree: GuildsTreeTheme{
			AutoExpandFolders:   true,
			ChannelColor:        tview.Styles.PrimaryTextColor.String(),
			Graphics:            true,
			GuildColor:          tview.Styles.PrimaryTextColor.String(),
			PrivateChannelColor: tview.Styles.PrimaryTextColor.String(),
		},
		MessagesText: MessagesTextTheme{
			ReplyIndicator: string(tview.BoxDrawingsLightArcDownAndRight) + " ",

			AuthorColor:     "aqua",
			ContentColor:    tview.Styles.PrimaryTextColor.String(),
			EmojiColor:      "green",
			LinkColor:       "blue",
			AttachmentColor: "yellow",
		},
	}
}
