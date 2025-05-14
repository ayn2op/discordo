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
	Timestamps struct {
		Enabled bool   `toml:"enabled"`
		Format  string `toml:"format"`
	}

	Identify struct {
		Status         discord.Status `toml:"status"`
		Browser        string         `toml:"browser"`
		BrowserVersion string         `toml:"browser_version"`
		UserAgent      string         `toml:"user_agent"`
	}

	Notifications struct {
		Enabled  bool  `toml:"enabled"`
		Duration int   `toml:"duration"`
		Sound    Sound `toml:"sound"`
	}

	Sound struct {
		Enabled    bool `toml:"enabled"`
		OnlyOnPing bool `toml:"only_on_ping"`
	}

	Config struct {
		Mouse  bool   `toml:"mouse"`
		Editor string `toml:"editor"`

		HideBlockedUsers    bool  `toml:"hide_blocked_users"`
		ShowAttachmentLinks bool  `toml:"show_attachment_links"`
		MessagesLimit       uint8 `toml:"messages_limit"`

		Timestamps    Timestamps    `toml:"timestamps"`
		Identify      Identify      `toml:"identify"`
		Notifications Notifications `toml:"notifications"`

		Keys  Keys  `toml:"keys"`
		Theme Theme `toml:"theme"`
	}
)

func defaultConfig() *Config {
	return &Config{
		Mouse:  true,
		Editor: "default",

		HideBlockedUsers:    true,
		ShowAttachmentLinks: true,
		MessagesLimit:       50,

		Timestamps: Timestamps{
			Enabled: true,
			Format:  time.Kitchen,
		},

		Identify: Identify{
			Status:         discord.OnlineStatus,
			Browser:        consts.Browser,
			BrowserVersion: consts.BrowserVersion,
			UserAgent:      consts.UserAgent,
		},

		Notifications: Notifications{
			Enabled:  true,
			Duration: 500,
			Sound: Sound{
				Enabled:    true,
				OnlyOnPing: true,
			},
		},

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

	path = filepath.Join(path, consts.Name, fileName)
	f, err := os.Open(path)

	cfg := defaultConfig()
	if os.IsNotExist(err) {
	    slog.Info("config does not exist, creating default config", "path", path)
	    f, err := os.Create(path)
	    if err == nil {
		toml.NewEncoder(f).Encode(cfg)
		f.Close()
	    }
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
