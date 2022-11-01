package config

import (
	_ "embed"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

const Name = "discordo"

type MessagesViewKeysConfig struct {
	OpenActionsView string `yaml:"open_actions_view"`

	SelectPreviousMessage string `yaml:"select_previous_message"`
	SelectNextMessage     string `yaml:"select_next_message"`
	SelectFirstMessage    string `yaml:"select_first_message"`
	SelectLastMessage     string `yaml:"select_last_message"`
}

type InputViewKeysConfig struct {
	OpenExternalEditor string `yaml:"open_external_editor"`
	PasteClipboard     string `yaml:"paste_clipboard"`
}

type KeysConfig struct {
	MessagesView MessagesViewKeysConfig `yaml:"messages_view"`
	InputView    InputViewKeysConfig    `yaml:"input_view"`
}

type ThemeConfig struct {
	Background string `yaml:"background"`
	Border     string `yaml:"border"`
	Title      string `yaml:"title"`
}

type Config struct {
	// Whether to send desktop notification when a new message is sent or not.
	Notifications bool `yaml:"notifications"`
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
	// Keybindings
	Keys KeysConfig `yaml:"keys"`
	// Theme
	Theme ThemeConfig `yaml:"theme"`
}

func New() *Config {
	return &Config{
		Notifications: true,
		Mouse:         true,
		MessagesLimit: 50,

		Timestamps: false,
		Timezone:   "Local",
		TimeFormat: time.Kitchen,

		Keys: KeysConfig{
			MessagesView: MessagesViewKeysConfig{
				OpenActionsView: "Rune[a]",

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

func (cfg *Config) Load(path string) error {
	// Open the existing configuration file with read-only flag.
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, os.ModePerm)
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
		return yaml.NewEncoder(f).Encode(cfg)
	}

	return yaml.NewDecoder(f).Decode(&cfg)
}

// DefaultConfigPath returns the default configuration file path.
func DefaultConfigPath() string {
	path, _ := os.UserConfigDir()
	return filepath.Join(path, Name+".yml")
}

// DefaultLogPath returns the default log file path.
func DefaultLogPath() string {
	path, _ := os.UserCacheDir()
	return filepath.Join(path, Name+".log")
}
