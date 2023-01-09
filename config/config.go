package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const Name = "discordo"

type Config struct {
	Mouse         bool   `yaml:"mouse"`
	MessagesLimit uint   `yaml:"messages_limit"`
	Timestamps    bool   `yaml:"timestamps"`
	Editor        string `yaml:"editor"`

	Keys  KeysConfig  `yaml:"keys"`
	Theme ThemeConfig `yaml:"theme"`
}

func New() (*Config, error) {
	path, err := os.UserConfigDir()
	if err != nil {
		return nil, err
	}

	path = filepath.Join(path, Name)
	err = os.MkdirAll(path, os.ModePerm)
	if err != nil {
		return nil, err
	}

	c := Config{
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
