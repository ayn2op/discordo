package config

import (
	"os"
	"path/filepath"

	"github.com/ayn2op/discordo/internal/constants"
	"gopkg.in/yaml.v3"
)

var Current = defConfig()

type Config struct {
	// Mouse indicates whether the mouse is usable or not.
	Mouse bool `yaml:"mouse"`
	// MessagesLimit is the number of messages to fetch when a text-based channel is selected.
	MessagesLimit uint `yaml:"messages_limit"`
	// TimestampsBeforeAuthor indicates whether to draw the timestamp before or after the author.
	TimestampsBeforeAuthor bool `yaml:"timestamps_before_author"`
	// Timestamps indicates whether to draw the timestamp in front of the message or not.
	Timestamps bool `yaml:"timestamps"`
	// Editor is the program to open when the `LaunchEditor` key is pressed. If the value of the field is "default", the `$EDITOR` environment variable is used instead.
	Editor string `yaml:"editor"`

	Keys  Keys  `yaml:"keys"`
	Theme Theme `yaml:"theme"`
}

func defConfig() Config {
	return Config{
		Mouse:                  true,
		TimestampsBeforeAuthor: false,
		Timestamps:             false,
		MessagesLimit:          50,
		Editor:                 "default",

		Keys:  defKeys(),
		Theme: defTheme(),
	}
}

func Load(path string) error {
	_, err := os.Stat(path)
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

	return nil
}

func DefaultPath() string {
	path, _ := os.UserConfigDir()
	return filepath.Join(path, constants.Name, "config.yml")
}

func DefaultLogPath() string {
	path, _ := os.UserCacheDir()
	return filepath.Join(path, constants.Name, "logs.txt")
}
