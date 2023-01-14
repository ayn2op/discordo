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

	Keys  Keys  `yaml:"keys"`
	Theme Theme `yaml:"theme"`
}

func new() Config {
	return Config{
		Mouse:         true,
		Timestamps:    false,
		MessagesLimit: 50,
		Editor:        "default",

		Keys:  newKeys(),
		Theme: newTheme(),
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
