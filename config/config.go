package config

import (
	_ "embed"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

const Name = "discordo"

type (
	MessagesTextKeysConfig struct {
		LaunchActions string `yaml:"launch_actions"`

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
		MessagesText MessagesTextKeysConfig `yaml:"messages_text"`
		MessageInput MessageInputKeysConfig `yaml:"message_input"`
	}
)

type ThemeConfig struct {
	Background string `yaml:"background"`
	Border     string `yaml:"border"`
	Title      string `yaml:"title"`
}

type Config struct {
	// Whether the mouse is usable or not.
	Mouse bool `yaml:"mouse"`
	// The maximum number of messages to fetch and display. Its value must not be lesser than 1 and greater than 100.
	MessagesLimit uint `yaml:"messages_limit"`
	// Whether to display the timestamps of the messages beside the displayed message or not.
	Timestamps bool `yaml:"timestamps"`
	// The timezone of the timestamps. Learn more: https://pkg.go.dev/time#LoadLocation
	Timezone string `yaml:"timezone"`
	// A textual representation of the time value formatted according to the layout defined by its value. Learn more: https://pkg.go.dev/time#Layout
	TimeFormat string `yaml:"time_format"`

	Keys  KeysConfig  `yaml:"keys"`
	Theme ThemeConfig `yaml:"theme"`
}

func New() (*Config, error) {
	path, err := os.UserConfigDir()
	if err != nil {
		return nil, err
	}

	// Create the configuration directory if it does not exist already.
	path = filepath.Join(path, Name)
	err = os.MkdirAll(path, os.ModePerm)
	if err != nil {
		return nil, err
	}

	c := def()
	path = filepath.Join(path, "config.yml")
	_, err = os.Stat(path)
	// If the configuration file does not exist, create a new one and write the default configuration to it.
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

func def() Config {
	return Config{
		Mouse:         true,
		MessagesLimit: 50,
		Timestamps:    false,
		Timezone:      "Local",
		TimeFormat:    time.Kitchen,

		Keys: KeysConfig{
			MessagesText: MessagesTextKeysConfig{
				LaunchActions: "Rune[a]",

				SelectPrevious: "Up",
				SelectNext:     "Down",
				SelectFirst:    "Home",
				SelectLast:     "End",
			},
			MessageInput: MessageInputKeysConfig{
				Send:  "Enter",
				Paste: "Ctrl+V",

				LaunchEditor: "Ctrl+E",
			},
		},
		Theme: ThemeConfig{
			Background: "default",
			Border:     "white",
			Title:      "white",
		},
	}
}

// LogDirPath returns the path of the log directory.
func LogDirPath() string {
	path, _ := os.UserCacheDir()
	return filepath.Join(path, Name)
}
