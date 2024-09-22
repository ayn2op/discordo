package config

import (
	"os"
	"path/filepath"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/ayn2op/discordo/internal/constants"
)

type Config struct {
	Mouse            bool `toml:"mouse"`
	HideBlockedUsers bool `toml:"hide_blocked_users"`

	MessagesLimit uint8  `toml:"messages_limit"`
	Editor        string `toml:"editor"`

	Timestamps             bool   `toml:"timestamps"`
	TimestampsBeforeAuthor bool   `toml:"timestamps_before_author"`
	TimestampsFormat       string `toml:"timestamps_format"`

	ShowAttachmentLinks bool `toml:"show_attachment_links"`

	Keys  Keys  `toml:"keys"`
	Theme Theme `toml:"theme"`
}

func defaultConfig() *Config {
	return &Config{
		Mouse:            true,
		HideBlockedUsers: true,
		MessagesLimit:    50,
		Editor:           "default",

		Timestamps:             false,
		TimestampsBeforeAuthor: false,
		TimestampsFormat:       time.Kitchen,
	
		ShowAttachmentLinks: true,

		Keys:  defaultKeys(),
		Theme: defaultTheme(),
	}
}

// Reads the configuration file and parses it.
func Load() (*Config, error) {
	path := filepath.Join(constants.ConfigDirPath, "config.toml")
	f, err := os.Open(path)

	cfg := defaultConfig()
	if os.IsNotExist(err) {
		return cfg, nil
	}

	if err != nil {
		return nil, err
	}
	defer f.Close()

	if _, err := toml.NewDecoder(f).Decode(&cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}
