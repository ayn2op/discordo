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
	Notifications    bool        `json:"notifications"`
	UserAgent        string      `json:"userAgent"`
	GetMessagesLimit int         `json:"getMessagesLimit"`
	Theme            tview.Theme `json:"theme"`
}

// LoadConfig creates (if the configuration file does not exist) and reads the configuration file and returns a new config.
func LoadConfig() *Config {
	u, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}

	configPath := u + "/.config/discordo/config.json"
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		f, err := os.Create(configPath)
		if err != nil {
			panic(err)
		}

		c := Config{
			Mouse:         true,
			Notifications: true,
			UserAgent: "" +
				"Mozilla/5.0 (X11; Linux x86_64) " +
				"AppleWebKit/537.36 (KHTML, like Gecko) " +
				"Chrome/92.0.4515.131 Safari/537.36",
			GetMessagesLimit: 50,
			Theme:            tview.Styles,
		}
		d, err := json.MarshalIndent(c, "", "\t")
		if err != nil {
			panic(err)
		}

		_, err = f.Write(d)
		if err != nil {
			panic(err)
		}

		f.Sync()
	}

	d, err := os.ReadFile(configPath)
	if err != nil {
		panic(err)
	}

	var c Config
	if err = json.Unmarshal(d, &c); err != nil {
		panic(err)
	}

	return &c
}
