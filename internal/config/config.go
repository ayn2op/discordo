package config

import (
	_ "embed"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"github.com/ayn2op/discordo/internal/consts"
)

const fileName = "config.toml"

type Config struct {
	Keybinds Keybinds `toml:"keybinds"`
}

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

	return &cfg, nil
}
