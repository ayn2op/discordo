package config

import "github.com/rivo/tview"

type (
	Theme struct {
		Border        bool   `toml:"border"`
		BorderColor   string `toml:"border_color"`
		BorderPadding [4]int `toml:"border_padding"`

		TitleColor      string `toml:"title_color"`
		BackgroundColor string `toml:"background_color"`

		GuildsTree   GuildsTreeTheme   `toml:"guilds_tree"`
		MessagesText MessagesTextTheme `toml:"messages_text"`
	}

	GuildsTreeTheme struct {
		AutoExpandFolders bool `toml:"auto_expand_folders"`
		Graphics          bool `toml:"graphics"`
	}

	MessagesTextTheme struct {
		AuthorColor    string `toml:"author_color"`
		ReplyIndicator string `toml:"reply_indicator"`
	}
)

func defaultTheme() Theme {
	return Theme{
		Border:        true,
		BorderColor:   "default",
		BorderPadding: [...]int{0, 0, 1, 1},

		BackgroundColor: "default",
		TitleColor:      "default",

		GuildsTree: GuildsTreeTheme{
			AutoExpandFolders: true,
			Graphics:          true,
		},
		MessagesText: MessagesTextTheme{
			AuthorColor:    "aqua",
			ReplyIndicator: string(tview.BoxDrawingsLightArcDownAndRight) + " ",
		},
	}
}
