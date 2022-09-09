package config

import (
	_ "embed"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

const Name = "discordo"

type MessagesPanelKeysConfig struct {
	OpenActionsList string `yaml:"open_actions_list"`

	SelectPreviousMessage string `yaml:"select_previous_message"`
	SelectNextMessage     string `yaml:"select_next_message"`
	SelectFirstMessage    string `yaml:"select_first_message"`
	SelectLastMessage     string `yaml:"select_last_message"`
}

type MessageInputKeysConfig struct {
	OpenExternalEditor string `yaml:"open_external_editor"`
	PasteClipboard     string `yaml:"paste_clipboard"`
}

type KeysConfig struct {
	MessagesPanel MessagesPanelKeysConfig `yaml:"messages_panel"`
	MessageInput  MessageInputKeysConfig  `yaml:"message_input"`
}

type ThemeConfig struct {
	Background string `yaml:"background"`
	Border     string `yaml:"border"`
	Title      string `yaml:"title"`
}

type Config struct {
	// Whether the mouse is usable or not.
	Mouse bool `yaml:"mouse"`
	// The maximum number of messages to fetch and display on the messages panel. Its value must not be lesser than 1 and greater than 100.
	MessagesLimit uint `yaml:"messages_limit"`
	// Whether to display the timestamps of the messages beside the displayed message or not.
	Timestamps bool `yaml:"timestamps"`
	// The timezone of the timestamps. Learn more: https://pkg.go.dev/time#LoadLocation
	Timezone string `yaml:"timezone"`
	// A textual representation of the time value formatted according to the layout defined by its value. Learn more: https://pkg.go.dev/time#Layout
	TimeFormat string `yaml:"time_format"`
	// Keybindings
	Keys KeysConfig `yaml:"keys"`
	// Theme
	Theme ThemeConfig `yaml:"theme"`
}

func New() *Config {
	return &Config{
		Mouse:         true,
		MessagesLimit: 50,

		Timestamps: false,
		Timezone:   "Local",
		TimeFormat: time.Kitchen,

		Keys: KeysConfig{
			MessagesPanel: MessagesPanelKeysConfig{
				OpenActionsList: "Rune[a]",

				SelectPreviousMessage: "Up",
				SelectNextMessage:     "Down",
				SelectFirstMessage:    "Home",
				SelectLastMessage:     "End",
			},
		},
		Theme: ThemeConfig{
			Background: "default",
			Border:     "white",
			Title:      "white",
		},
	}
}

func (c *Config) Load() error {
	configPath, err := os.UserConfigDir()
	if err != nil {
		return err
	}

	configPath = filepath.Join(configPath, Name)
	// Create directories that do not exist and are mentioned in the path recursively.
	err = os.MkdirAll(configPath, os.ModePerm)
	if err != nil {
		return err
	}

	configPath = filepath.Join(configPath, "config.yml")
	// Open the existing configuration file with read-only flag.
	f, err := os.OpenFile(configPath, os.O_CREATE|os.O_RDWR, os.ModePerm)
	if err != nil {
		return err
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		return err
	}

	// If the configuration file is empty (the size of the file is zero; a new configuration file was created), write the default configuration to the file.
	if fi.Size() == 0 {
		return yaml.NewEncoder(f).Encode(c)
	}

	return yaml.NewDecoder(f).Decode(&c)
}
