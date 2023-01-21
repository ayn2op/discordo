package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const Name = "discordo"

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

// Load reads the configuration file and decodes the configuration file or creates a new one if it does not exist already and writes the default configuration to the newly-created configuration file.
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

	c := Config{
		Mouse:         true,
		Timestamps:    false,
		MessagesLimit: 50,
		Editor:        "default",

		Keys:  newKeys(),
		Theme: newTheme(),
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
