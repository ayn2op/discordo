package util

import (
	"encoding/json"
	"os"

	"github.com/rivo/tview"
)

// Config consists of fields, such as theme, mouse, so on, that may be customized by the user.
type Config struct {
	Token            string       `json:"token,omitempty"`
	Mouse            bool         `json:"mouse,omitempty"`
	GetMessagesLimit int          `json:"getMessagesLimit,omitempty"`
	Theme            *tview.Theme `json:"theme,omitempty"`
}

// NewConfig reads the configuration file (if exists) and returns a new config.
func NewConfig() *Config {
	c := Config{
		Mouse:            true,
		GetMessagesLimit: 50,
	}

	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		return &c
	}
	configPath := userHomeDir + "/.config/discordo/config.json"
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return &c
	}

	d, err := os.ReadFile(configPath)
	if err != nil {
		panic(err)
	}

	if err = json.Unmarshal(d, &c); err != nil {
		panic(err)
	}

	return &c
}
