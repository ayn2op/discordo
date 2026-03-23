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
		// Whether to send typing status or not.
		Send bool `toml:"send"`
		// Whether to receive typing status or not.
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
		// Whether to parse and render markdown in messages or not.
		Enabled bool `toml:"enabled"`
		// The theme for fenced code blocks.
		// Available themes: https://xyproto.github.io/splash/docs.
		Theme string `toml:"theme"`
	}

	HelpConfig struct {
		// Show compact key modifiers in help, e.g. "^x" instead of "ctrl+x".
		CompactModifiers bool `toml:"compact_modifiers"`
		// The horizontal padding around help content: [left, right].
		Padding [2]int `toml:"padding"`
		// The visual separator between keybinds.
		Separator string `toml:"separator"`
	}

	SidebarMarkersConfig struct {
		Expanded  string `toml:"expanded"`
		Collapsed string `toml:"collapsed"`
		Leaf      string `toml:"leaf"`
	}

	SidebarConfig struct {
		Markers SidebarMarkersConfig `toml:"markers"`
	}
)

//go:generate go run ./cmd/generate.go
type Config struct {
	// Whether to focus the message input automatically when a channel is selected.
	// Set to false to preview channels without moving focus.
	AutoFocus bool `toml:"auto_focus"`
	// Whether to enable mouse or not.
	Mouse bool `toml:"mouse"`
	// The program to open when the `message_input.editor` keybind is pressed.
	// Set it to an empty value to use `$EDITOR` environment variable.
	Editor string `toml:"editor"`
	// "default" (unknown), "online", "dnd", "idle", "invisible", "offline"
	Status discord.Status `toml:"status"`

	HideBlockedUsers    bool `toml:"hide_blocked_users"`
	ShowAttachmentLinks bool `toml:"show_attachment_links"`

	// The maximum number of members to populate in the mentions list.
	// Set to 0 to disable.
	AutocompleteLimit uint8 `toml:"autocomplete_limit"`
	// The number of messages to fetch when a text-based channel is selected from guilds tree.
	// The minimum and maximum value is 1 and 100, respectively.
	MessagesLimit uint8 `toml:"messages_limit"`

	Markdown        MarkdownConfig  `toml:"markdown"`
	Help            HelpConfig      `toml:"help"`
	Picker          PickerConfig    `toml:"picker"`
	Timestamps      Timestamps      `toml:"timestamps"`
	DateSeparator   DateSeparator   `toml:"date_separator"`
	Notifications   Notifications   `toml:"notifications"`
	TypingIndicator TypingIndicator `toml:"typing_indicator"`
	Sidebar         SidebarConfig   `toml:"sidebar"`

	Icons Icons `toml:"icons"`
	// Each keybind field accepts either a string or a list of strings.
	// Type: `Keybind = string or []string`.
	// ```toml
	// [keybinds]
	// quit = "ctrl+q"
	// # or,
	// quit = ["ctrl+q", "ctrl+c", "q", ...]
	// ```
	Keybinds Keybinds `toml:"keybinds"`
	// Types:
	// Alignment = "left" | "center" | "right"
	// Attributes = string | []string
	// Style = { foreground?, background?, attributes?, underline?, underline_color? }
	// BorderSet = "hidden" | "plain" | "round" | "thick" | "double"
	// GlyphSet = "minimal" | "box_drawing" | "boxdrawing" | "box" | "unicode"
	// ScrollBarVisibility = "automatic" | "auto" | "always" | "never" | "hidden" | "off"
	Theme Theme `toml:"theme"`
}

func defaultConfig() Config {
	return Config{
		AutoFocus: true,
		Mouse:     true,
		Editor:    "default",
		Status:    "default",

		HideBlockedUsers:    true,
		ShowAttachmentLinks: true,

		AutocompleteLimit: 20,
		MessagesLimit:     50,

		Markdown: MarkdownConfig{
			Enabled: true,
			Theme:   "monokai",
		},
		Help: HelpConfig{
			CompactModifiers: true,
			Padding:          [2]int{1, 1},
			Separator:        " • ",
		},
		Picker: PickerConfig{
			Width:  80,
			Height: 25,
		},
		Timestamps: Timestamps{
			Enabled: true,
			Format:  "3:04PM",
		},
		DateSeparator: DateSeparator{
			Enabled:   true,
			Format:    "January 2, 2006",
			Character: "─",
		},
		Notifications: Notifications{
			Enabled:  true,
			Duration: 0,
			Sound: Sound{
				Enabled:    true,
				OnlyOnPing: true,
			},
		},
		TypingIndicator: TypingIndicator{
			Send:    true,
			Receive: true,
		},
		Sidebar: SidebarConfig{
			Markers: SidebarMarkersConfig{
				Expanded:  "▾ ",
				Collapsed: "▸ ",
				Leaf:      "",
			},
		},

		Icons: Icons{
			GuildCategory:           "",
			GuildText:               "#",
			GuildVoice:              "♪ ",
			GuildStageVoice:         "♪ ",
			GuildAnnouncementThread: "a-",
			GuildPublicThread:       "› ",
			GuildPrivateThread:      "› ",
			GuildAnnouncement:       "a-",
			GuildForum:              "≡ ",
			GuildStore:              "s-",
		},
		Keybinds: defaultKeybinds(),
		Theme:    defaultTheme(),
	}
}

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
	cfg := defaultConfig()

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

	if cfg.DateSeparator.Format == "" {
		cfg.DateSeparator.Format = "January 2, 2006"
	}
	if cfg.DateSeparator.Character == "" {
		cfg.DateSeparator.Character = "─"
		return
	}

	r, _ := utf8.DecodeRuneInString(cfg.DateSeparator.Character)
	if r == utf8.RuneError {
		cfg.DateSeparator.Character = "─"
		return
	}
	cfg.DateSeparator.Character = string(r)
}
