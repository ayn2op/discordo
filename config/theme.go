package config

type (
	GuildsTreeTheme struct {
		// Graphics indicates whether to draw the line graphics to illustrate the hierarchy or not.
		Graphics bool `yaml:"graphics"`
	}

	MessagesTextTheme struct {
		AuthorColor string `yaml:"author_color"`
		ReplyIndicator string `yaml:"reply_indicator"`
	}

	MessageInputTheme struct{}
)

type Theme struct {
	Border        bool   `yaml:"border"`
	BorderColor   string `yaml:"border_color"`
	BorderPadding [4]int `yaml:"border_padding,flow"`

	TitleColor      string `yaml:"title_color"`
	BackgroundColor string `yaml:"background_color"`

	GuildsTree   GuildsTreeTheme   `yaml:"guilds_tree"`
	MessagesText MessagesTextTheme `yaml:"messages_text"`
	MessageInput MessageInputTheme `yaml:"message_input"`
}

func defTheme() Theme {
	return Theme{
		Border:        true,
		BorderColor:   "default",
		BorderPadding: [...]int{0, 0, 1, 1},

		TitleColor:      "default",
		BackgroundColor: "default",

		GuildsTree: GuildsTreeTheme{
			Graphics: true,
		},
		MessagesText: MessagesTextTheme{
			AuthorColor: "aqua",
			ReplyIndicator: "╭",
		},
		MessageInput: MessageInputTheme{},
	}
}
