package config

import (
	"os"
	"path/filepath"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/ayn2op/discordo/internal/constants"
)

type Config struct {
	Mouse bool `toml:"mouse"`

	Timestamps             bool `toml:"timestamps"`
	TimestampsBeforeAuthor bool `toml:"timestamps_before_author"`
	TimestampsFormat       string `toml:"timestamps_format"`

	MessagesLimit uint8 `toml:"messages_limit"`

	Editor string `toml:"editor"`

	Keys  Keys  `toml:"keys"`
	Theme Theme `toml:"theme"`
}

func DefaultConfig() Config {
	return Config{
		Mouse: true,

		Timestamps:             false,
		TimestampsBeforeAuthor: false,
		TimestampsFormat:       time.Kitchen,

		MessagesLimit: 50,
		Editor:        "default",

		Keys:  defaultKeys(),
		Theme: defaultTheme(),
	}
}

// Reads the configuration file and parses it.
func Load() (*Config, error) {
	path, err := os.UserConfigDir()
	if err != nil {
		return nil, err
	}

	cfg := DefaultConfig()
	path = filepath.Join(path, constants.Name, "config.toml")
	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return &cfg, nil
	}

	if err != nil {
		return nil, err
	}
	defer f.Close()

	if _, err := toml.NewDecoder(f).Decode(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
