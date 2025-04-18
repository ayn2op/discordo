package config

import (
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/ayn2op/discordo/internal/consts"
	"github.com/diamondburned/arikawa/v3/discord"
)

const fileName = "config.toml"

type (
	Identify struct {
		Status         discord.Status `toml:"status"`
		Browser        string         `toml:"browser"`
		BrowserVersion string         `toml:"browser_version"`
		UserAgent      string         `toml:"user_agent"`
	}

	Config struct {
		Mouse  bool   `toml:"mouse"`
		Editor string `toml:"editor"`

		HideBlockedUsers    bool  `toml:"hide_blocked_users"`
		ShowAttachmentLinks bool  `toml:"show_attachment_links"`
		MessagesLimit       uint8 `toml:"messages_limit"`

		Timestamps       bool   `toml:"timestamps"`
		TimestampsFormat string `toml:"timestamps_format"`

		Identify      Identify      `toml:"identify"`
		Keys          Keys          `toml:"keys"`
		Theme         Theme         `toml:"theme"`
		Notifications Notifications `toml:"notifications"`
	}
)

func defaultConfig() *Config {
	return &Config{
		Mouse:  true,
		Editor: "default",

		HideBlockedUsers:    true,
		ShowAttachmentLinks: true,
		MessagesLimit:       50,

		Timestamps:       false,
		TimestampsFormat: time.Kitchen,

		Identify: Identify{
			Status:         discord.OnlineStatus,
			Browser:        consts.Browser,
			BrowserVersion: consts.BrowserVersion,
			UserAgent:      consts.UserAgent,
		},

		Keys:          defaultKeys(),
		Theme:         defaultTheme(),
		Notifications: defaultNotifications(),
	}
}

// Reads the configuration file and parses it.
func Load() (*Config, error) {
	path, err := os.UserConfigDir()
	if err != nil {
		slog.Info("user configuration directory path cannot be determined; falling back to the current directory path")
		path = "."
	}

	path = filepath.Join(path, consts.Name, fileName)
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
