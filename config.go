package main

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const name = "discordo"

type CommonThemeConfig struct {
	Border        bool   `yaml:"border"`
	BorderPadding [4]int `yaml:"border_padding,flow"`
}

type GuildsTreeThemeConfig struct {
	CommonThemeConfig `yaml:",inline"`
	Graphics          bool `yaml:"graphics"`
}

type MessagesTextThemeConfig struct {
	CommonThemeConfig `yaml:",inline"`
}

type MessageInputThemeConfig struct {
	CommonThemeConfig `yaml:",inline"`
}

type ThemeConfig struct {
	GuildsTree   GuildsTreeThemeConfig   `yaml:"guilds_tree"`
	MessagesText MessagesTextThemeConfig `yaml:"messages_text"`
	MessageInput MessageInputThemeConfig `yaml:"message_input"`
}

type Config struct {
	Mouse         bool `yaml:"mouse"`
	MessagesLimit uint `yaml:"messages_limit"`

	Theme ThemeConfig `yaml:"theme"`
}

func newConfig() (*Config, error) {
	path, err := os.UserConfigDir()
	if err != nil {
		return nil, err
	}

	path = filepath.Join(path, name)
	if err = os.MkdirAll(path, os.ModePerm); err != nil {
		return nil, err
	}

	common := CommonThemeConfig{
		Border:        true,
		BorderPadding: [...]int{1, 1, 1, 1},
	}

	c := Config{
		Mouse:         true,
		MessagesLimit: 50,

		Theme: ThemeConfig{
			GuildsTree: GuildsTreeThemeConfig{
				CommonThemeConfig: common,
				Graphics:          true,
			},
			MessagesText: MessagesTextThemeConfig{
				CommonThemeConfig: common,
			},
			MessageInput: MessageInputThemeConfig{
				CommonThemeConfig: common,
			},
		},
	}
	path = filepath.Join(path, "config.yaml")
	if _, err = os.Stat(path); os.IsNotExist(err) {
		f, err := os.Create(path)
		if err != nil {
			return nil, err
		}
		defer f.Close()

		e := yaml.NewEncoder(f)
		if err = e.Encode(c); err != nil {
			return nil, err
		}
	} else {
		f, err := os.Open(path)
		if err != nil {
			return nil, err
		}
		defer f.Close()

		if err = yaml.NewDecoder(f).Decode(&c); err != nil {
			return nil, err
		}
	}

	return &c, nil
}
