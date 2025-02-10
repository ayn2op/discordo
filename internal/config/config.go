package config

import (
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/ayn2op/discordo/internal/consts"
)

type Config struct {
	Mouse               bool   `toml:"mouse"`
	HideBlockedUsers    bool   `toml:"hide_blocked_users"`
	Timestamps          bool   `toml:"timestamps"`
	ShowAttachmentLinks bool   `toml:"show_attachment_links"`
	MessagesLimit       uint8  `toml:"messages_limit"`
	Editor              string `toml:"editor"`

	Browser        string `toml:"browser"`
	BrowserVersion string `toml:"browser_version"`
	UserAgent      string `toml:"user_agent"`

	TimestampsFormat string `toml:"timestamps_format"`
	Keys             Keys   `toml:"keys"`
	Theme            Theme  `toml:"theme"`
}

func defaultConfig() *Config {
	return &Config{
		Mouse:               true,
		HideBlockedUsers:    true,
		Timestamps:          false,
		ShowAttachmentLinks: true,
		MessagesLimit:       50,
		Editor:              "default",
		TimestampsFormat:    time.Kitchen,

		Browser:        consts.Browser,
		BrowserVersion: consts.BrowserVersion,
		UserAgent:      consts.UserAgent,

		Keys:  defaultKeys(),
		Theme: defaultTheme(),
	}
}

// Reads the configuration file and parses it.
func Load() (*Config, error) {
	path, err := os.UserConfigDir()
	if err != nil {
		slog.Info("user configuration directory path cannot be determined; falling back to the current directory path")
		path = "."
	}

	path = filepath.Join(path, consts.Name, "config.toml")
	f, err := os.Open(path)

	cfg := defaultConfig()
	if os.IsNotExist(err) {
		slog.Info("the configuration file does not exist, falling back to the default configuration", "path", path, "err", err)
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
