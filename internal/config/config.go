package config

//go:generate go run ./cmd/generate.go

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/BurntSushi/toml"
	"github.com/ayn2op/discordo/internal/consts"
	"github.com/diamondburned/arikawa/v3/discord"
)

const fileName = "config.toml"

type Status discord.Status

var _ toml.Unmarshaler = (*Status)(nil)

func (s *Status) UnmarshalTOML(value any) error {
	str, ok := value.(string)
	if !ok {
		return errInvalidType
	}

	switch str {
	case "unknown":
		*s = Status(discord.UnknownStatus)
	case "online":
		*s = Status(discord.OnlineStatus)
	case "dnd":
		*s = Status(discord.DoNotDisturbStatus)
	case "idle":
		*s = Status(discord.IdleStatus)
	case "invisible":
		*s = Status(discord.InvisibleStatus)
	case "offline":
		*s = Status(discord.OfflineStatus)
	default:
		return fmt.Errorf("invalid status: %q", str)
	}
	return nil
}

type (
	Timestamps struct {
		Enabled bool   `toml:"enabled" default:"true"`
		Format  string `toml:"format" default:"3:04PM"`
	}

	DateSeparator struct {
		Enabled   bool   `toml:"enabled" default:"true"`
		Format    string `toml:"format" default:"January 2, 2006"`
		Character string `toml:"character" default:"─"`
	}

	Notifications struct {
		Enabled  bool  `toml:"enabled" default:"true"`
		Duration int   `toml:"duration" default:"0"`
		Sound    Sound `toml:"sound"`
	}

	Sound struct {
		Enabled    bool `toml:"enabled" default:"true"`
		OnlyOnPing bool `toml:"only_on_ping" default:"true"`
	}

	TypingIndicator struct {
		// Whether to send typing status or not.
		Send bool `toml:"send" default:"true"`
		// Whether to receive typing status or not.
		Receive bool `toml:"receive" default:"true"`
	}

	Icons struct {
		GuildCategory   string `toml:"guild_category" default:""`
		GuildText       string `toml:"guild_text" default:"#"`
		GuildVoice      string `toml:"guild_voice" default:"♪ "`
		GuildStageVoice string `toml:"guild_stage_voice" default:"♪ "`

		GuildAnnouncementThread string `toml:"guild_announcement_thread" default:"a-"`
		GuildPublicThread       string `toml:"guild_public_thread" default:"› "`
		GuildPrivateThread      string `toml:"guild_private_thread" default:"› "`

		GuildAnnouncement string `toml:"guild_announcement" default:"a-"`
		GuildForum        string `toml:"guild_forum" default:"≡ "`
		GuildStore        string `toml:"guild_store" default:"s-"`
	}

	PickerConfig struct {
		Width  int `toml:"width" default:"80"`
		Height int `toml:"height" default:"25"`
	}

	MarkdownConfig struct {
		// Whether to parse and render markdown in messages or not.
		Enabled bool `toml:"enabled" default:"true"`
		// The theme for fenced code blocks.
		// Available themes: https://xyproto.github.io/splash/docs.
		Theme string `toml:"theme" default:"monokai"`
	}

	HelpConfig struct {
		// Show compact key modifiers in help, e.g. "^x" instead of "ctrl+x".
		CompactModifiers bool `toml:"compact_modifiers" default:"true"`
		// The horizontal padding around help content: [left, right].
		Padding [2]int `toml:"padding" default:"[1, 1]"`
		// The visual separator between keybinds.
		Separator string `toml:"separator" default:" • "`
	}

	SidebarMarkersConfig struct {
		Expanded  string `toml:"expanded" default:"▾ "`
		Collapsed string `toml:"collapsed" default:"▸ "`
		Leaf      string `toml:"leaf" default:""`
	}

	SidebarConfig struct {
		Markers SidebarMarkersConfig `toml:"markers"`
	}
)

type Config struct {
	// Whether to focus the message input automatically when a channel is selected.
	// Set to false to preview channels without moving focus.
	AutoFocus bool `toml:"auto_focus" default:"true"`
	// Whether to enable mouse or not.
	Mouse bool `toml:"mouse" default:"true"`
	// The program to open when the `message_input.editor` keybind is pressed.
	// Set it to `"default"` to use `$EDITOR` environment variable.
	Editor string `toml:"editor" default:"default"`
	// Values: "unknown", "online", "dnd", "idle", "invisible", "offline"
	Status Status `toml:"status" default:"unknown"`

	HideBlockedUsers    bool `toml:"hide_blocked_users" default:"true"`
	ShowAttachmentLinks bool `toml:"show_attachment_links" default:"true"`

	// The maximum number of members to populate in the mentions list.
	// Set to 0 to disable.
	AutocompleteLimit uint8 `toml:"autocomplete_limit" default:"20"`
	// The number of messages to fetch when a text-based channel is selected from guilds tree.
	// The minimum and maximum value is 1 and 100, respectively.
	MessagesLimit uint8 `toml:"messages_limit" default:"50"`

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
	if err := applyTagDefaults(&cfg); err != nil {
		return nil, fmt.Errorf("failed to apply config defaults: %w", err)
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

var tomlUnmarshalerType = reflect.TypeFor[toml.Unmarshaler]()
var keybindType = reflect.TypeFor[Keybind]()

func applyTagDefaults(cfg *Config) error {
	return applyTagDefaultsValue(reflect.ValueOf(cfg).Elem())
}

func applyTagDefaultsValue(v reflect.Value) error {
	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		fv := v.Field(i)
		if !fv.CanSet() {
			continue
		}
		if fv.Kind() == reflect.Struct && !implementsTOMLUnmarshaler(fv.Type()) {
			if err := applyTagDefaultsValue(fv); err != nil {
				return err
			}
		}
		def, ok := field.Tag.Lookup("default")
		if ok {
			if err := decodeDefaultValue(fv, def); err != nil {
				return fmt.Errorf("%s.%s: %w", t.Name(), field.Name, err)
			}
		}
		if help, ok := field.Tag.Lookup("help"); ok && help != "" {
			applyKeybindHelp(fv, help)
		}
	}
	return nil
}

func applyKeybindHelp(v reflect.Value, help string) {
	if v.Type() != keybindType {
		return
	}
	k := v.Addr().Interface().(*Keybind)
	if keys := k.Keys(); len(keys) > 0 {
		k.SetHelp(keys[0], help)
	}
}

func decodeDefaultValue(v reflect.Value, expr string) error {
	if expr == "" {
		if v.Kind() != reflect.String {
			return fmt.Errorf("empty default is only supported for string-like types")
		}
		v.Set(reflect.Zero(v.Type()))
		return nil
	}

	parse := func(input string) error {
		// Decode as TOML so custom UnmarshalTOML types are handled automatically.
		typeHolder := reflect.StructOf([]reflect.StructField{{
			Name: "V",
			Type: v.Type(),
			Tag:  `toml:"v"`,
		}})
		holder := reflect.New(typeHolder)
		if _, err := toml.Decode("v = "+input, holder.Interface()); err != nil {
			return err
		}
		v.Set(holder.Elem().Field(0))
		return nil
	}

	if v.Kind() == reflect.String {
		return parse(strconv.Quote(expr))
	}
	err := parse(expr)
	if err == nil {
		return nil
	}
	if implementsTOMLUnmarshaler(v.Type()) && !isQuoted(expr) {
		return parse(strconv.Quote(expr))
	}
	return err
}

func implementsTOMLUnmarshaler(t reflect.Type) bool {
	return t.Implements(tomlUnmarshalerType) || reflect.PointerTo(t).Implements(tomlUnmarshalerType)
}

func isQuoted(s string) bool {
	s = strings.TrimSpace(s)
	return (strings.HasPrefix(s, "'") && strings.HasSuffix(s, "'")) ||
		(strings.HasPrefix(s, "\"") && strings.HasSuffix(s, "\""))
}

func applyDefaults(cfg *Config) {
	if cfg.Editor == "default" {
		cfg.Editor = os.Getenv("EDITOR")
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
