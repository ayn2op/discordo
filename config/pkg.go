package config

import (
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

type Config struct {
	Token  string `toml:"-" name:"token" help:"The authentication token." short:"T"`
	Config string `toml:"-" name:"config" help:"The path of the configuration file." type:"path" short:"C"`

	General GeneralConfig `toml:"general" kong:"-"`
	Theme   ThemeConfig   `toml:"theme" kong:"-"`
	Keys    KeysConfig    `toml:"keys" kong:"-"`
}

func New() *Config {
	return &Config{
		Config: DefaultPath(),

		General: newGeneralConfig(),
		Theme:   newThemeConfig(),
		Keys:    newKeysConfig(),
	}
}

func (c *Config) Load() {
	err := os.MkdirAll(filepath.Dir(c.Config), os.ModePerm)
	if err != nil {
		panic(err)
	}

	if _, err = os.Stat(c.Config); os.IsNotExist(err) {
		f, err := os.Create(c.Config)
		if err != nil {
			panic(err)
		}

		err = toml.NewEncoder(f).Encode(c)
		if err != nil {
			panic(err)
		}
	} else {
		_, err = toml.DecodeFile(c.Config, &c)
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
