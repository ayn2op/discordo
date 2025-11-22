package config

import (
	_ "embed"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"github.com/ayn2op/discordo/config"
	"github.com/ayn2op/discordo/internal/consts"
	"github.com/diamondburned/arikawa/v3/discord"
)

const fileName = "config.toml"

type (
	Timestamps struct {
		Enabled bool   `toml:"enabled" default:"true" description:"https://pkg.go.dev/time#Layout"`
		Format  string `toml:"format" default:"3:04PM"`
	}

	Notifications struct {
		Enabled  bool  `toml:"enabled" default:"true"`
		Duration int   `toml:"duration" default:"0" description:"The duration of the sound. Set 0 to use default duration. This is only supported on Unix and Windows."`
		Sound    Sound `toml:"sound"`
	}

	Sound struct {
		Enabled    bool `toml:"enabled" default:"true"`
		OnlyOnPing bool `toml:"only_on_ping" default:"true"`
	}

	Config struct {
		Mouse  bool   `toml:"mouse" default:"true" description:"Whether to enable mouse support or not."`
		Editor string `toml:"editor" default:"vim" env:"EDITOR" description:"The program to open when the keys.message_input.open_editor is pressed."`

		Status discord.Status `toml:"status" default:"unknown"`

		Markdown            bool `toml:"markdown" default:"true" description:"Whether to parse and render markdown or not."`
		HideBlockedUsers    bool `toml:"hide_blocked_users" default:"true"`
		ShowAttachmentLinks bool `toml:"show_attachment_links" default:"true"`

		AutocompleteLimit uint8 `toml:"autocomplete_limit" default:"20"`
		MessagesLimit     uint8 `toml:"messages_limit" default:"50" description:"The number of messages to fetch when a text-based channel is selected. The minimum and maximum value is 0 and 100, respectively."`

		Timestamps    Timestamps    `toml:"timestamps"`
		Notifications Notifications `toml:"notifications"`

		Keys  Keys  `toml:"keys"`
		Theme Theme `toml:"theme"`
	}
)

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
	if err := config.Load(&cfg); err != nil {
		return nil, err
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

	return &cfg, nil
}
