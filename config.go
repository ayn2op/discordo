package main

import (
	"encoding/json"
	"os"
	"path/filepath"
)

const name = "discordo"

type ThemeConfig struct {
	BorderPadding [4]int
}

type GuildsTreeKeysConfig struct {
	Graphics bool
}

type MessagesTextKeysConfig struct {
	SelectPreviousMessage string
}

type KeysConfig struct {
	GuildsTree   GuildsTreeKeysConfig
	MessagesText MessagesTextKeysConfig
}

type Config struct {
	Mouse         bool
	MessagesLimit uint

	Theme ThemeConfig
	Keys  KeysConfig
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

	c := Config{
		Mouse:         true,
		MessagesLimit: 50,

		Theme: ThemeConfig{
			BorderPadding: [...]int{1, 1, 1, 1},
		},
		Keys: KeysConfig{
			GuildsTree:   GuildsTreeKeysConfig{Graphics: true},
			MessagesText: MessagesTextKeysConfig{SelectPreviousMessage: "Up"},
		},
	}
	path = filepath.Join(path, "config.json")
	if _, err = os.Stat(path); os.IsNotExist(err) {
		f, err := os.Create(path)
		if err != nil {
			return nil, err
		}
		defer f.Close()

		e := json.NewEncoder(f)
		e.SetIndent("", "\t")
		if err = e.Encode(c); err != nil {
			return nil, err
		}
	} else {
		f, err := os.Open(path)
		if err != nil {
			return nil, err
		}
		defer f.Close()

		if err = json.NewDecoder(f).Decode(&c); err != nil {
			return nil, err
		}
	}

	return &c, nil
}

func (c *Config) BorderPadding() (int, int, int, int) {
	pad := c.Theme.BorderPadding
	return pad[0], pad[1], pad[2], pad[3]
}
