package config

import (
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

type Config struct {
	General     GeneralConfig     `toml:"general"`
	Theme       ThemeConfig       `toml:"theme"`
	Keybindings KeybindingsConfig `toml:"keybindings"`
}

func New() *Config {
	return &Config{
		General:     newGeneralConfig(),
		Theme:       newThemeConfig(),
		Keybindings: newKeybindingsConfig(),
	}
}

func (c *Config) Load(path string) {
	err := os.MkdirAll(filepath.Dir(path), os.ModePerm)
	if err != nil {
		panic(err)
	}

	if _, err = os.Stat(path); os.IsNotExist(err) {
		f, err := os.Create(path)
		if err != nil {
			panic(err)
		}

		err = toml.NewEncoder(f).Encode(c)
		if err != nil {
			panic(err)
		}
	} else {
		_, err = toml.DecodeFile(path, &c)
		if err != nil {
			panic(err)
		}
	}
}

func DefaultPath() string {
	path, err := os.UserConfigDir()
	if err != nil {
		panic(err)
	}

	path += "/discordo/config.toml"
	return path
}
