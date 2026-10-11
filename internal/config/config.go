package config

import (
	"fmt"
	"log/slog"
	"mime"
	"os"
	"path"
	"path/filepath"
	"slices"
	"sync"
	"unicode/utf8"

	"github.com/BurntSushi/toml"
	"github.com/ayn2op/arikawa/v3/discord"
	"github.com/ayn2op/discordo/internal/consts"
)

const fileName = "config.toml"

type MIMETypes []string

func (types *MIMETypes) UnmarshalTOML(value any) error {
	values, ok := value.([]any)
	if !ok {
		return errInvalidType
	}

	parsed := make(MIMETypes, 0, len(values))
	for _, value := range values {
		mediaType, ok := value.(string)
		if !ok {
			return errInvalidType
		}
		mediaType, _, err := mime.ParseMediaType(mediaType)
		if err != nil {
			return err
		}
		parsed = append(parsed, mediaType)
	}

	*types = parsed
	return nil
}

func (types MIMETypes) Has(mediaType string) bool {
	mediaType, _, err := mime.ParseMediaType(mediaType)
	if err != nil {
		return false
	}

	return slices.ContainsFunc(types, func(pattern string) bool {
		matches, _ := path.Match(pattern, mediaType)
		return matches
	})
}

type (
	Timestamps struct {
		Enabled bool `toml:"enabled"`
		// https://pkg.go.dev/time#Layout
		Format string `toml:"format"`
	}

	AttachmentsConfig struct {
		ShowLinks bool `toml:"show_links"`
		// Attachment MIME types that can be opened or downloaded without confirmation.
		// Exact types and wildcards such as "image/*" are supported.
		AllowedMIMETypes MIMETypes `toml:"allowed_mime_types"`
		// Whether to draw image attachments and embedded images below their messages.
		Preview bool `toml:"preview"`
		// How previews are drawn: "auto", "kitty" or "halfblocks".
		Protocol string `toml:"protocol"`
	}

	DateSeparator struct {
		Enabled bool `toml:"enabled"`
		// https://pkg.go.dev/time#Layout
		Format string `toml:"format"`
		// The fill character used on both sides of the date label.
		Character string `toml:"character"`
	}

	Notifications struct {
		Enabled bool `toml:"enabled"`
		// Whether to also notify about messages in the open channel while the terminal window is unfocused.
		// Requires a terminal that reports focus.
		WhenUnfocused bool `toml:"when_unfocused"`
	}

	TypingIndicator struct {
		// Whether to send typing status.
		Send bool `toml:"send"`
		// Whether to receive typing status.
		Receive bool `toml:"receive"`
	}

	Voice struct {
		// How loud the microphone must be for what is said to be sent, in dB below the loudest it can be.
		Sensitivity float64 `toml:"sensitivity"`
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

		// Shown after the members of a voice channel who are muted or deafened.
		VoiceMuted    string `toml:"voice_muted"`
		VoiceDeafened string `toml:"voice_deafened"`
	}

	PickerConfig struct {
		Width  int `toml:"width"`
		Height int `toml:"height"`
	}

	MarkdownConfig struct {
		// Whether to parse and render markdown in messages.
		Enabled bool `toml:"enabled"`
		// Replace spoiler contents with a placeholder.
		MaskSpoilers bool `toml:"mask_spoilers"`
		// Theme for fenced code blocks. Available themes: https://xyproto.github.io/splash/docs
		Theme string `toml:"theme"`
	}

	HelpConfig struct {
		// Whether to show keybind help.
		Enabled bool `toml:"enabled"`
		// Show compact key modifiers in help, e.g. "^x" instead of "ctrl+x".
		CompactModifiers bool `toml:"compact_modifiers"`
		// [left, right]
		Padding   [2]int `toml:"padding"`
		Separator string `toml:"separator"`
	}

	ComposerConfig struct {
		// Maximum height (in newline-separated rows) the composer grows to before it starts scrolling internally.
		// Set to 1 for a fixed single-line input. Values below 1 fall back to 10.
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

	// PaneConfig is a side pane that can be toggled at runtime.
	PaneConfig struct {
		// Whether the pane is visible at start-up. It can still be toggled at runtime with its toggle keybind.
		Visible bool `toml:"visible"`
		// Percentage (%) of the available width used by the pane.
		// Valid values are 1-99. Invalid values fall back to 20.
		WidthPercent int `toml:"width_percent"`
	}

	SidebarConfig struct {
		PaneConfig
		Markers SidebarMarkersConfig `toml:"markers"`
		// Indentation for different item types.
		Indents SidebarIndentsConfig `toml:"indents"`
	}

	Config struct {
		// Whether to focus the composer automatically when a channel is selected.
		// Set to false to preview channels without moving focus.
		AutoFocus bool `toml:"auto_focus"`
		// Whether to enable the mouse.
		Mouse bool `toml:"mouse"`
		// The program opened by the `composer.open_editor` keybind. "default" uses `$EDITOR`.
		Editor string `toml:"editor"`

		// "default" (unknown), "online", "dnd", "idle", "invisible" or "offline".
		Status           discord.Status `toml:"status"`
		HideBlockedUsers bool           `toml:"hide_blocked_users"`

		// The maximum number of members in the mention autocomplete list. 0 disables it.
		AutocompleteLimit uint8 `toml:"autocomplete_limit"`
		// The number of messages to fetch when a text-based channel is selected, from 1 to 100.
		MessagesLimit uint8 `toml:"messages_limit"`

		Attachments AttachmentsConfig `toml:"attachments"`

		Markdown        MarkdownConfig  `toml:"markdown"`
		Help            HelpConfig      `toml:"help"`
		Picker          PickerConfig    `toml:"picker"`
		Timestamps      Timestamps      `toml:"timestamps"`
		DateSeparator   DateSeparator   `toml:"date_separator"`
		Notifications   Notifications   `toml:"notifications"`
		TypingIndicator TypingIndicator `toml:"typing_indicator"`
		// The guilds tree pane.
		Sidebar SidebarConfig `toml:"sidebar"`
		// The members tree pane, sized from the width right of the guilds tree.
		MembersTree PaneConfig     `toml:"members_tree"`
		Composer    ComposerConfig `toml:"composer"`

		// Layout and behavior of the widgets.
		UI    UIConfig `toml:"ui"`
		Voice Voice    `toml:"voice"`
		Icons Icons    `toml:"icons"`

		// Global shortcuts. A keybind is a key such as "ctrl+k", or a list of keys.
		Keybinds Keybinds `toml:"keybinds"`
		// Styles of the widgets, each written as
		// `{ foreground = "", background = "", attributes = "" or ["", ""], underline = "", underline_color = "" }`.
		Theme ThemeConfig `toml:"theme"`
	}
)

//go:generate go run gen.go

// Default returns the default configuration.
func Default() Config {
	pane := PaneConfig{Visible: true, WidthPercent: 20}
	return Config{
		AutoFocus: true,
		Mouse:     true,
		Editor:    "default",

		Status:           "default",
		HideBlockedUsers: true,

		AutocompleteLimit: 20,
		MessagesLimit:     50,

		Attachments: AttachmentsConfig{
			ShowLinks:        true,
			AllowedMIMETypes: MIMETypes{"image/*", "video/*"},
			Preview:          true,
			Protocol:         "auto",
		},

		Markdown:        MarkdownConfig{Enabled: true, Theme: "monokai"},
		Help:            HelpConfig{Enabled: true, CompactModifiers: true, Padding: [2]int{1, 1}, Separator: " • "},
		Picker:          PickerConfig{Width: 80, Height: 25},
		Timestamps:      Timestamps{Enabled: true, Format: "3:04PM"},
		DateSeparator:   DateSeparator{Enabled: true, Format: "January 2, 2006", Character: "─"},
		Notifications:   Notifications{Enabled: true, WhenUnfocused: true},
		TypingIndicator: TypingIndicator{Send: true, Receive: true},
		Sidebar: SidebarConfig{
			PaneConfig: pane,
			Markers:    SidebarMarkersConfig{Expanded: "▾ ", Collapsed: "▸ "},
			Indents:    SidebarIndentsConfig{Guild: 2, Category: 1, Channel: 2, Forum: 2, GroupDM: 1, DM: 2},
		},
		MembersTree: pane,
		Composer:    ComposerConfig{MaxHeight: 10},

		UI:    defaultUI(),
		Voice: Voice{Sensitivity: -40},
		Icons: Icons{
			GuildText:       "#",
			GuildVoice:      "♪ ",
			GuildStageVoice: "♪ ",

			GuildAnnouncementThread: "a-",
			GuildPublicThread:       "› ",
			GuildPrivateThread:      "› ",

			GuildAnnouncement: "a-",
			GuildForum:        "≡ ",
			GuildStore:        "s-",

			VoiceMuted:    " (muted)",
			VoiceDeafened: " (deafened)",
		},

		Keybinds: defaultKeybinds(),
		Theme:    defaultTheme(),
	}
}

var dir = sync.OnceValue(func() string {
	path, err := os.UserConfigDir()
	if err != nil {
		slog.Info("user config dir cannot be determined; falling back to the current dir", "err", err)
		path = "."
	}
	return filepath.Join(path, consts.Name)
})

func DefaultPath() string {
	return filepath.Join(dir(), fileName)
}

// Load reads the configuration file and parses it.
func Load(path string) (*Config, error) {
	cfg := Default()
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

	cfg.applyDefaults()
	return &cfg, nil
}

func (cfg *Config) applyDefaults() {
	if cfg.Editor == "default" {
		cfg.Editor = os.Getenv("EDITOR")
	}

	if cfg.Status == "default" {
		cfg.Status = discord.UnknownStatus
	}

	if cfg.Composer.MaxHeight <= 0 {
		cfg.Composer.MaxHeight = 10
	}

	for _, pane := range []*PaneConfig{&cfg.Sidebar.PaneConfig, &cfg.MembersTree} {
		if pane.WidthPercent <= 0 || pane.WidthPercent >= 100 {
			pane.WidthPercent = 20
		}
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
