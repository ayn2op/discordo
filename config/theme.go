package config

type (
	GuildsTreeThemeConfig struct {
		Graphics bool `yaml:"graphics"`
	}

	MessagesTextThemeConfig struct {
		AuthorColor string `yaml:"author_color"`
	}

	MessageInputThemeConfig struct{}

	ThemeConfig struct {
		Border        bool   `yaml:"border"`
		BorderColor   string `yaml:"border_color"`
		BorderPadding [4]int `yaml:"border_padding,flow"`

		TitleColor      string `yaml:"title_color"`
		BackgroundColor string `yaml:"background_color"`

		GuildsTree   GuildsTreeThemeConfig   `yaml:"guilds_tree"`
		MessagesText MessagesTextThemeConfig `yaml:"messages_text"`
		MessageInput MessageInputThemeConfig `yaml:"message_input"`
	}
)
