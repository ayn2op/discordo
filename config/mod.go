package config

import (
	"os"

	"github.com/BurntSushi/toml"
)

type GeneralConfig struct {
	UserAgent          string `json:"userAgent"`
	FetchMessagesLimit int    `json:"fetchMessagesLimit"`
	Mouse              bool   `json:"mouse"`
	Notifications      bool   `json:"notifications"`
	Timestamps         bool   `json:"timestamps"`
}

type KeybindingsConfig struct {
	FocusGuildsList         string `json:"focusGuildsList"`
	FocusChannelsTreeView   string `json:"focusChannelsTreeView"`
	FocusMessagesTextView   string `json:"focusMessagesTextView"`
	FocusMessageInputField  string `json:"focusMessageInputField"`
	FocusMessageActionsList string `json:"focusMessageActionsList"`

	OpenEditor string `json:"open_editor"`

	SelectPreviousMessage string `json:"selectPreviousMessage"`
	SelectNextMessage     string `json:"selectNextMessage"`
	SelectFirstMessage    string `json:"selectFirstMessage"`
	SelectLastMessage     string `json:"selectLastMessage"`
}

type Config struct {
	Keybindings KeybindingsConfig `json:"keybindings"`
	General     GeneralConfig     `json:"general"`
}

func Load() Config {
	configPath, err := os.UserConfigDir()
	if err != nil {
		panic(err)
	}

	configPath += "/discordo.toml"
	c := Config{}
	// If the configuration file does not exist, create and write the default configuration to the file.
	if _, err = os.Stat(configPath); os.IsNotExist(err) {
		f, err := os.Create(configPath)
		if err != nil {
			panic(err)
		}
		defer f.Close()

		c = newDefaultConfig()
		err = toml.NewEncoder(f).Encode(c)
		if err != nil {
			panic(err)
		}
	} else {
		_, err = toml.DecodeFile(configPath, &c)
		if err != nil {
			panic(err)
		}
	}

	return c
}

func newDefaultConfig() Config {
	return Config{
		General: GeneralConfig{
			UserAgent:          "Mozilla/5.0 (X11; Linux x86_64; rv:95.0) Gecko/20100101 Firefox/95.0",
			FetchMessagesLimit: 50,
			Mouse:              true,
			Notifications:      true,
			Timestamps:         false,
		},
		Keybindings: KeybindingsConfig{
			FocusGuildsList:         "Alt+Rune[g]",
			FocusChannelsTreeView:   "Alt+Rune[t]",
			FocusMessagesTextView:   "Alt+Rune[m]",
			FocusMessageInputField:  "Alt+Rune[i]",
			FocusMessageActionsList: "Alt+Rune[a]",

			OpenEditor: "Alt+Rune[e]",

			SelectPreviousMessage: "Up",
			SelectNextMessage:     "Down",
			SelectFirstMessage:    "Home",
			SelectLastMessage:     "End",
		},
	}
}
