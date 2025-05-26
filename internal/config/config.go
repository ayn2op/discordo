package config

import (
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

		MarkdownEnabled bool `toml:"markdown_enabled"`

		Timestamps    Timestamps    `toml:"timestamps"`
		Identify      Identify      `toml:"identify"`
		Notifications Notifications `toml:"notifications"`

		Keys  Keys  `toml:"keys"`
		Theme Theme `toml:"theme"`
	}
)

//go:embed config.toml
var defaultCfg []byte

func DefaultPath() string {
	path, err := os.UserConfigDir()
	if err != nil {
		slog.Info("user configuration directory path cannot be determined; falling back to the current directory path")
		path = "."
	}

	return filepath.Join(path, consts.Name, fileName)
}

// Reads the configuration file and parses it.
func Load(path string) (*Config, error) {
	f, err := os.Open(path)

	var cfg *Config
	if err := toml.Unmarshal(defaultCfg, &cfg); err != nil {
		return nil, err
	}

	if os.IsNotExist(err) {
		slog.Info("the configuration file does not exist, falling back to the default configuration", "path", path, "err", err)
		handleDefaults(cfg)
		return cfg, nil
	}

	if err != nil {
		return nil, err
	}
	defer f.Close()

	if _, err := toml.NewDecoder(f).Decode(&cfg); err != nil {
		return nil, err
	}

	handleDefaults(cfg)
	return cfg, nil
}

func handleDefaults(cfg *Config) {
	if cfg.Identify.Browser == "default" {
		cfg.Identify.Browser = consts.Browser
	}
	if cfg.Identify.BrowserVersion == "default" {
		cfg.Identify.BrowserVersion = consts.BrowserVersion
	}
	if cfg.Identify.UserAgent == "default" {
		cfg.Identify.UserAgent = consts.UserAgent
	}
}
