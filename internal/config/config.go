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

		Markdown            bool `toml:"markdown"`
		HideBlockedUsers    bool `toml:"hide_blocked_users"`
		ShowAttachmentLinks bool `toml:"show_attachment_links"`

		// Use 0 to disable
		AutocompleteLimit uint8 `toml:"autocomplete_limit"`
		MessagesLimit     uint8 `toml:"messages_limit"`

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
		slog.Info(
			"user configuration directory path cannot be determined; falling back to the current directory path",
		)
		path = "."
	}

	return filepath.Join(path, consts.Name, fileName)
}

// Load reads the configuration file and parses it.
func Load(path string) (*Config, error) {
	file, err := os.Open(path)

	var cfg *Config
	if err := toml.Unmarshal(defaultCfg, &cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal default config: %w", err)
	}

	if os.IsNotExist(err) {
		slog.Info(
			"the configuration file does not exist, falling back to the default configuration",
			"path",
			path,
			"err",
			err,
		)
		handleDefaults(cfg)
		return cfg, nil
	}

	if err != nil {
		return nil, fmt.Errorf("failed to open config file: %w", err)
	}
	defer file.Close()

	if _, err := toml.NewDecoder(file).Decode(&cfg); err != nil {
		return nil, fmt.Errorf("failed to decode config: %w", err)
	}

	handleDefaults(cfg)
	return cfg, nil
}

func handleDefaults(cfg *Config) {
	if cfg.Editor == "default" {
		cfg.Editor = os.Getenv("EDITOR")
	}

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

func (cfg *Config) GetMemberName(m *discord.Member) string {
	if cfg.Theme.MessagesText.ShowNicknames && m.Nick != "" {
		return m.Nick
	}
	return cfg.GetUserName(&m.User)
}

func (cfg *Config) GetUserName(u *discord.User) string {
	if cfg.Theme.MessagesText.ShowUsernames || u.DisplayName == "" {
		return u.Username
	}
	return u.DisplayName
}
