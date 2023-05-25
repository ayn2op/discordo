package config

import (
	"errors"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const Name = "discordo"

var Current = defConfig()

type Config struct {
	// Mouse indicates whether the mouse is usable or not.
	Mouse bool `yaml:"mouse"`
	// MessagesLimit is the number of messages to fetch when a text-based channel is selected.
	MessagesLimit uint `yaml:"messages_limit"`
	// Timestamps indicates whether to draw the timestamp after the author name or before it.
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

func getPath(optionalPath string) (string, error) {
	// Trigger an error if config flag used but is empty.
	if optionalPath == "" {
		return "", errors.New("Optional path cannot be empty.")
	}

	// Use the path provided by flags.
	if optionalPath != "none" {
		return optionalPath, nil
	}

	// Use the default for the OS.
	path, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}

	path = filepath.Join(path, Name)
	if err != nil {
		return "", err
	}

	path = filepath.Join(path, "config.yml")
	if err != nil {
		return "", err
	}

	return path, nil
}

func Load(optionalPath string) error {
	path, err := getPath(optionalPath)
	if err != nil {
		return err
	}

	// Split the directory from the configuration file.
	dir, file := filepath.Split(path)

	// Create the configuration directory if it does not exist already.
	err = os.MkdirAll(dir, os.ModePerm)
	if err != nil {
		return err
	}

	path = filepath.Join(dir, file)
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

	return nil
}
