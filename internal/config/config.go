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

	Icons struct {
		GuildCategory   string `toml:"guild_category"`
		GuildText       string `toml:"guild_text"`
		GuildVoice      string `toml:"guild_voice"`
		GuildStageVoice string `toml:"guild_stage_voice"`

		GuildAnnouncementThread string `toml:"guild_announcement_thread"`
		GuildPublicThread       string `toml:"guild_public_thread"`
		GuildPrivateThread      string `toml:"guild_private_thread"`

		GuildAnnouncement string `toml:"guild_announcement"`
		GuildForum        string `toml:"guild_forum"`
		GuildStore        string `toml:"guild_store"`
	}

	PickerConfig struct {
		Width  int `toml:"width"`
		Height int `toml:"height"`
	}

	Config struct {
		AutoFocus bool   `toml:"auto_focus"`
		Mouse     bool   `toml:"mouse"`
		Editor    string `toml:"editor"`

		Status discord.Status `toml:"status"`

		Markdown            bool   `toml:"markdown"`
		HideBlockedUsers    bool   `toml:"hide_blocked_users"`
		ShowAttachmentLinks bool   `toml:"show_attachment_links"`
		ShowSpoiler         string `toml:"show_spoiler"`

		// Use 0 to disable
		AutocompleteLimit uint8 `toml:"autocomplete_limit"`
		MessagesLimit     uint8 `toml:"messages_limit"`

		Picker          PickerConfig    `toml:"picker"`
		Timestamps      Timestamps      `toml:"timestamps"`
		Notifications   Notifications   `toml:"notifications"`
		TypingIndicator TypingIndicator `toml:"typing_indicator"`

		Icons Icons `toml:"icons"`

		Keybinds Keybinds `toml:"keybinds"`
		Theme    Theme    `toml:"theme"`
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
