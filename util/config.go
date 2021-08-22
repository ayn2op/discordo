package util

import (
	"encoding/json"
	"os"
)

// Theme defines the theme for the application.
type Theme struct {
	Background string `json:"background,omitempty"`
	Foreground string `json:"foreground,omitempty"`
	Borders    bool   `json:"borders,omitempty"`
}

// Config consists of fields, such as theme, mouse, so on, that may be customized by the user.
type Config struct {
	Token            string `json:"token,omitempty"`
	Mouse            bool   `json:"mouse,omitempty"`
	GetMessagesLimit int    `json:"getMessagesLimit,omitempty"`
	Theme            *Theme `json:"theme,omitempty"`
}

// NewConfig reads the configuration file (if exists) and returns a new config.
func NewConfig() *Config {
	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}

	var c Config = Config{
		Mouse:            true,
		GetMessagesLimit: 50,
		Theme: &Theme{
			Borders: true,
		},
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
