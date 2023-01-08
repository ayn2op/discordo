package main

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const name = "discordo"

type (
	CommonKeysConfig struct {
		Cancel string `yaml:"cancel"`
	}

	MessagesTextKeysConfig struct {
		CommonKeysConfig `yaml:",inline"`
		Reply            string `yaml:"reply"`
		ReplyMention     string `yaml:"reply_mention"`
	}

	MessageInputKeysConfig struct {
		CommonKeysConfig `yaml:",inline"`
		Send             string `yaml:"send"`
		LaunchEditor     string `yaml:"launch_editor"`
	}

	KeysConfig struct {
		MessagesText MessagesTextKeysConfig `yaml:"messages_text"`
		MessageInput MessageInputKeysConfig `yaml:"message_input"`
	}
)

type (
	CommonThemeConfig struct {
		Border        bool   `yaml:"border"`
		BorderPadding [4]int `yaml:"border_padding,flow"`

		TitleColor      string `yaml:"title_color"`
		BackgroundColor string `yaml:"background_color"`
	}

	GuildsTreeThemeConfig struct {
		CommonThemeConfig `yaml:",inline"`
		Graphics          bool `yaml:"graphics"`
	}

	MessagesTextThemeConfig struct {
		CommonThemeConfig `yaml:",inline"`
		AuthorColor       string `yaml:"author_color"`
	}

	MessageInputThemeConfig struct {
		CommonThemeConfig `yaml:",inline"`
	}

	ThemeConfig struct {
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

func newConfig() (*Config, error) {
	path, err := os.UserConfigDir()
	if err != nil {
		return nil, err
	}

	path = filepath.Join(path, name)
	err = os.MkdirAll(path, os.ModePerm)
	if err != nil {
		return nil, err
	}

	commonTheme := CommonThemeConfig{
		Border:        true,
		BorderPadding: [...]int{0, 0, 1, 1},

		TitleColor:      "default",
		BackgroundColor: "default",
	}

	commonKeys := CommonKeysConfig{
		Cancel: "Esc",
	}

	c := Config{
		Mouse:         true,
		Timestamps:    false,
		MessagesLimit: 50,
		Editor:        "default",

		Keys: KeysConfig{
			MessagesText: MessagesTextKeysConfig{
				CommonKeysConfig: commonKeys,
				Reply:            "Rune[r]",
				ReplyMention:     "Rune[R]",
			},
			MessageInput: MessageInputKeysConfig{
				CommonKeysConfig: commonKeys,
				Send:             "Enter",
				LaunchEditor:     "Ctrl+E",
			},
		},

		Theme: ThemeConfig{
			GuildsTree: GuildsTreeThemeConfig{
				CommonThemeConfig: commonTheme,
				Graphics:          true,
			},
			MessagesText: MessagesTextThemeConfig{
				CommonThemeConfig: commonTheme,
				AuthorColor:       "aqua",
			},
			MessageInput: MessageInputThemeConfig{
				CommonThemeConfig: commonTheme,
			},
		},
	}
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
