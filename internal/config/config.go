package config

import (
	_ "embed"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

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

	Notifications struct {
		Enabled  bool  `toml:"enabled"`
		Duration int   `toml:"duration"`
		Sound    Sound `toml:"sound"`
	}

	Sound struct {
		Enabled    bool `toml:"enabled"`
		OnlyOnPing bool `toml:"only_on_ping"`
	}

	TypingIndicator struct {
		Send    bool `toml:"send"`
		Receive bool `toml:"receive"`
	}

	Config struct {
		AutoFocus bool   `toml:"auto_focus"`
		Mouse     bool   `toml:"mouse"`
		Editor    string `toml:"editor"`

		Status discord.Status `toml:"status"`

		Markdown            bool `toml:"markdown"`
		HideBlockedUsers    bool `toml:"hide_blocked_users"`
		ShowAttachmentLinks bool `toml:"show_attachment_links"`

		// Use 0 to disable
		AutocompleteLimit uint8 `toml:"autocomplete_limit"`
		MessagesLimit     uint8 `toml:"messages_limit"`

		Timestamps      Timestamps      `toml:"timestamps"`
		Notifications   Notifications   `toml:"notifications"`
		TypingIndicator TypingIndicator `toml:"typing_indicator"`

		Keys  Keys  `toml:"keys"`
		Theme Theme `toml:"theme"`
	}
)

//go:embed config.toml
var defaultCfg []byte

func DefaultPath() string {
	path, err := os.UserConfigDir()
	if err != nil {
		slog.Info(
			"user config dir cannot be determined; falling back to the current dir",
			"err", err,
		)
		path = "."
	}

	return filepath.Join(path, consts.Name, fileName)
}

// Load reads the configuration file and parses it.
func Load(path string) (*Config, error) {
	var cfg Config
	if err := toml.Unmarshal(defaultCfg, &cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal default config: %w", err)
	}

	file, err := os.Open(path)
	if os.IsNotExist(err) {
		slog.Info(
			"config file does not exist, falling back to the default config",
			"path",
			path,
			"err",
			err,
		)
	} else {
		if err != nil {
			return nil, fmt.Errorf("failed to open config file: %w", err)
		}
		defer file.Close()

		if _, err := toml.NewDecoder(file).Decode(&cfg); err != nil {
			return nil, fmt.Errorf("failed to decode config: %w", err)
		}
	}

	applyDefaults(&cfg)
	return &cfg, nil
}

func applyDefaults(cfg *Config) {
	if cfg.Editor == "default" {
		cfg.Editor = os.Getenv("EDITOR")
	}

	if cfg.Status == "default" {
		cfg.Status = discord.UnknownStatus
	}
}
