package util

import (
	"encoding/json"
	"os"

	"github.com/rivo/tview"
)

// Config consists of fields, such as theme, mouse, so on, that may be
// customized by the user.
type Config struct {
	Token            string      `json:"token"`
	Mouse            bool        `json:"mouse"`
	UserAgent        string      `json:"userAgent"`
	GetMessagesLimit int         `json:"getMessagesLimit"`
	Theme            tview.Theme `json:"theme"`
}

// NewConfig reads the configuration file (if exists) and returns a new config.
func NewConfig() *Config {
	c := Config{
		Token: "",
		Mouse: true,
		UserAgent: "" +
			"Mozilla/5.0 (X11; Linux x86_64) " +
			"AppleWebKit/537.36 (KHTML, like Gecko) " +
			"Chrome/92.0.4515.131 Safari/537.36",
		GetMessagesLimit: 50,
		Theme:            tview.Styles,
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
