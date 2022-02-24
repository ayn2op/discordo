package config

import (
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

type Config struct {
	Keybindings KeybindingsConfig `toml:"keybindings"`
	General     GeneralConfig     `toml:"general"`
}

func newConfig() Config {
	return Config{
		General:     newGeneralConfig(),
		Keybindings: newKeybindingsConfig(),
	}
}

func NewConfig() *Config {
	path, err := os.UserConfigDir()
	if err != nil {
		panic(err)
	}

	path += "/discordo/config.toml"

	err = os.MkdirAll(filepath.Dir(path), os.ModePerm)
	if err != nil {
		panic(err)
	}

	var c Config
	if _, err = os.Stat(path); os.IsNotExist(err) {
		f, err := os.Create(path)
		if err != nil {
			panic(err)
		}

		c = newConfig()
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

	return &c
}
