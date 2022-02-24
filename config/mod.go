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

	f, err := os.Open(path)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	var c Config
	if _, err = f.Stat(); os.IsNotExist(err) {
		f, err = os.Create(path)
		if err != nil {
			panic(err)
		}
		defer f.Close()

		c = newConfig()
		err = toml.NewEncoder(f).Encode(c)
		if err != nil {
			panic(err)
		}
	} else {
		_, err = toml.NewDecoder(f).Decode(&c)
		if err != nil {
			panic(err)
		}
	}

	return &c
}
