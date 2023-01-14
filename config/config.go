package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const Name = "discordo"

type (
	MessagesTextKeysConfig struct {
		CopyContent string `yaml:"copy_content"`

		Reply        string `yaml:"reply"`
		ReplyMention string `yaml:"reply_mention"`
		SelectReply  string `yaml:"select_reply"`

		SelectPrevious string `yaml:"select_previous"`
		SelectNext     string `yaml:"select_next"`
		SelectFirst    string `yaml:"select_first"`
		SelectLast     string `yaml:"select_last"`
	}

	MessageInputKeysConfig struct {
		Send  string `yaml:"send"`
		Paste string `yaml:"paste"`

		LaunchEditor string `yaml:"launch_editor"`
	}

	KeysConfig struct {
		Cancel string `yaml:"cancel"`

		MessagesText MessagesTextKeysConfig `yaml:"messages_text"`
		MessageInput MessageInputKeysConfig `yaml:"message_input"`
	}
)

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

type Config struct {
	Mouse         bool   `yaml:"mouse"`
	MessagesLimit uint   `yaml:"messages_limit"`
	Timestamps    bool   `yaml:"timestamps"`
	Editor        string `yaml:"editor"`

	Keys  KeysConfig  `yaml:"keys"`
	Theme ThemeConfig `yaml:"theme"`
}

func new() Config {
	return Config{
		Mouse:         true,
		Timestamps:    false,
		MessagesLimit: 50,
		Editor:        "default",

		Keys: KeysConfig{
			Cancel: "Esc",

			MessagesText: MessagesTextKeysConfig{
				CopyContent: "Rune[c]",

				Reply:        "Rune[r]",
				ReplyMention: "Rune[R]",
				SelectReply:  "Rune[s]",

				SelectPrevious: "Up",
				SelectNext:     "Down",
				SelectFirst:    "Home",
				SelectLast:     "End",
			},
			MessageInput: MessageInputKeysConfig{
				Send: "Enter",

				Paste:        "Ctrl+V",
				LaunchEditor: "Ctrl+E",
			},
		},

		Theme: ThemeConfig{
			Border:        true,
			BorderColor:   "default",
			BorderPadding: [...]int{0, 0, 1, 1},

			TitleColor:      "default",
			BackgroundColor: "default",

			GuildsTree: GuildsTreeThemeConfig{
				Graphics: true,
			},
			MessagesText: MessagesTextThemeConfig{
				AuthorColor: "aqua",
			},
			MessageInput: MessageInputThemeConfig{},
		},
	}
}

func Load() (*Config, error) {
	path, err := os.UserConfigDir()
	if err != nil {
		return nil, err
	}

	path = filepath.Join(path, Name)
	err = os.MkdirAll(path, os.ModePerm)
	if err != nil {
		return nil, err
	}

	c := new()
	path = filepath.Join(path, "config.yml")
	_, err = os.Stat(path)
	if os.IsNotExist(err) {
		f, err := os.Create(path)
		if err != nil {
			return nil, err
		}
		defer f.Close()

		err = yaml.NewEncoder(f).Encode(c)
		if err != nil {
			return nil, err
		}
	} else {
		f, err := os.Open(path)
		if err != nil {
			return nil, err
		}
		defer f.Close()

		err = yaml.NewDecoder(f).Decode(&c)
		if err != nil {
			return nil, err
		}
	}

	return &c, nil
}
