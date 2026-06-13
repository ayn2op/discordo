package config

import (
	_ "embed"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"unicode/utf8"

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

	DateSeparator struct {
		Enabled   bool   `toml:"enabled"`
		Format    string `toml:"format"`
		Character string `toml:"character"`
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

	MarkdownConfig struct {
		Enabled bool   `toml:"enabled"`
		Theme   string `toml:"theme"`
	}

	HelpConfig struct {
		CompactModifiers bool   `toml:"compact_modifiers"`
		Padding          [2]int `toml:"padding"`
		Separator        string `toml:"separator"`
	}

	ComposerConfig struct {
		// MaxHeight caps how tall (in newline-separated rows) the input grows before it starts scrolling internally.
		// Must be >= 1; values <= 0 fall back to the default.
		MaxHeight int `toml:"max_height"`
	}

	SidebarMarkersConfig struct {
		Expanded  string `toml:"expanded"`
		Collapsed string `toml:"collapsed"`
		Leaf      string `toml:"leaf"`
	}

	SidebarIndentsConfig struct {
		Guild    int `toml:"guild"`
		Category int `toml:"category"`
		Channel  int `toml:"channel"`
		Forum    int `toml:"forum"`
		GroupDM  int `toml:"group_dm"`
		DM       int `toml:"dm"`
	}

	SidebarConfig struct {
		// WidthPercent is the percentage (%) of the total
		// window width that the guilds tree sidebar occupies
		WidthPercent   int                  `toml:"width_percent"`
		Markers        SidebarMarkersConfig `toml:"markers"`
		Indents        SidebarIndentsConfig `toml:"indents"`
	}

	Config struct {
		AutoFocus bool   `toml:"auto_focus"`
		Mouse     bool   `toml:"mouse"`
		Editor    string `toml:"editor"`

		Status              discord.Status `toml:"status"`
		HideBlockedUsers    bool           `toml:"hide_blocked_users"`
		ShowAttachmentLinks bool           `toml:"show_attachment_links"`

		// Use 0 to disable
		AutocompleteLimit uint8 `toml:"autocomplete_limit"`
		MessagesLimit     uint8 `toml:"messages_limit"`

		Markdown        MarkdownConfig  `toml:"markdown"`
		Help            HelpConfig      `toml:"help"`
		Picker          PickerConfig    `toml:"picker"`
		Timestamps      Timestamps      `toml:"timestamps"`
		DateSeparator   DateSeparator   `toml:"date_separator"`
		Notifications   Notifications   `toml:"notifications"`
		TypingIndicator TypingIndicator `toml:"typing_indicator"`
		Sidebar         SidebarConfig   `toml:"sidebar"`
		Composer        ComposerConfig  `toml:"composer"`

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
		slog.Info("user config dir cannot be determined; falling back to the current dir", "err", err)
		path = "."
	}

	return filepath.Join(path, consts.Name, fileName)
}

// Load reads the configuration file and parses it.
func Load(path string) (*Config, error) {
	cfg := Config{
		Keybinds: defaultKeybinds(),
	}
	if err := toml.Unmarshal(defaultCfg, &cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal default config: %w", err)
	}

	file, err := os.Open(path)
	switch {
	case os.IsNotExist(err):
		slog.Info("config file does not exist, falling back to the default config", "path", path, "err", err)
	case err != nil:
		return nil, fmt.Errorf("failed to open config file: %w", err)
	default:
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

	if cfg.Composer.MaxHeight <= 0 {
		cfg.Composer.MaxHeight = 10
	}

	if cfg.Sidebar.WidthPercent <= 0 || cfg.Sidebar.WidthPercent >= 100 {
		// these guidelines are simply to guarantee functionality;
		// there's no guarantee that there's functional utility in
		// setting an extremely low width, but that's for the
		// user to decide.
		cfg.Sidebar.WidthPercent = 20
	}

	if cfg.DateSeparator.Format == "" {
		cfg.DateSeparator.Format = "January 2, 2006"
	}
	if r, _ := utf8.DecodeRuneInString(cfg.DateSeparator.Character); r == utf8.RuneError {
		cfg.DateSeparator.Character = "─"
	} else {
		cfg.DateSeparator.Character = string(r)
	}
}
