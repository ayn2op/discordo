package config

import (
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

type Config struct {
	Path    string        `toml:"-"`
	General GeneralConfig `toml:"general"`
	Theme   ThemeConfig   `toml:"theme"`
	Keys    KeysConfig    `toml:"keys"`
}

func New() *Config {
	return &Config{
		Path:    DefaultPath(),
		General: newGeneralConfig(),
		Theme:   newThemeConfig(),
		Keys:    newKeysConfig(),
	}
}

func (c *Config) Load() {
	err := os.MkdirAll(filepath.Dir(c.Path), os.ModePerm)
	if err != nil {
		panic(err)
	}

	if _, err = os.Stat(c.Path); os.IsNotExist(err) {
		f, err := os.Create(c.Path)
		if err != nil {
			panic(err)
		}

		err = toml.NewEncoder(f).Encode(c)
		if err != nil {
			panic(err)
		}
	} else {
		_, err = toml.DecodeFile(c.Path, &c)
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
