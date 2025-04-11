package config

import (
	"github.com/rivo/tview"
)

type TitleAlign int

func (ta *TitleAlign) UnmarshalTOML(v any) error {
	switch v.(string) {
	case "left":
		*ta = tview.AlignLeft
	case "center":
		*ta = tview.AlignCenter
	case "right":
		*ta = tview.AlignRight
	}

	return nil
}

type (
	BorderTheme struct {
		Enabled bool   `toml:"enabled"`
		Padding [4]int `toml:"padding"`

		Color       string `toml:"color"`
		ActiveColor string `toml:"active_color"`

		Preset BorderPreset `toml:"preset"`
	}

	TitleTheme struct {
		Color       string     `toml:"color"`
		ActiveColor string     `toml:"active_color"`
		Align       TitleAlign `toml:"align"`
	}

	Theme struct {
		BackgroundColor string `toml:"background_color"`

		Title        TitleTheme        `toml:"title"`
		Border       BorderTheme       `toml:"border"`
		GuildsTree   GuildsTreeTheme   `toml:"guilds_tree"`
		MessagesText MessagesTextTheme `toml:"messages_text"`
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
		BackgroundColor: "default",

		Border: BorderTheme{
			Enabled: true,
			Padding: [...]int{0, 0, 1, 1},

			Color:       "default",
			ActiveColor: "green",
			Preset:      borderPresetRound(),
		},

		Title: TitleTheme{
			Color:       "default",
			ActiveColor: "green",
			Align:       tview.AlignLeft,
		},

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
