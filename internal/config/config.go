package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const Name = "discordo"

var Current = defConfig()

type Config struct {
	// Mouse indicates whether the mouse is usable or not.
	Mouse bool `yaml:"mouse"`
	// MessagesLimit is the number of messages to be retrieved when a text-based channel is selected.
	MessagesLimit uint `yaml:"messages_limit"`
	// Timestamps indicates whether the message is to be prefixed with the timestamp or not.
	Timestamps bool `yaml:"timestamps"`
	// Editor is the editor program to open when the `LaunchEditor` key is pressed. If the value of the field is "default", the `$EDITOR` environment variable is used instead.
	Editor string `yaml:"editor"`

	Keys  Keys  `yaml:"keys"`
	Theme Theme `yaml:"theme"`
}

func defConfig() Config {
	return Config{
		Mouse:         true,
		Timestamps:    false,
		MessagesLimit: 50,
		Editor:        "default",

		Keys:  defKeys(),
		Theme: defTheme(),
	}
}

func Load() error {
	path, err := os.UserConfigDir()
	if err != nil {
		return err
	}

	// Create the configuration directory if it does not exist already.
	path = filepath.Join(path, Name)
	err = os.MkdirAll(path, os.ModePerm)
	if err != nil {
		return err
	}

	path = filepath.Join(path, "config.yml")
	_, err = os.Stat(path)
	if os.IsNotExist(err) {
		f, err := os.Create(path)
		if err != nil {
			return err
		}
		defer f.Close()

		err = yaml.NewEncoder(f).Encode(Current)
		if err != nil {
			return err
		}
	} else {
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()

		err = yaml.NewDecoder(f).Decode(&Current)
		if err != nil {
			return err
		}
	}

	return err
}
