package config

import (
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

const userAgent = "Mozilla/5.0 (X11; Linux x86_64; rv:95.0) Gecko/20100101 Firefox/95.0"

type GeneralConfig struct {
	UserAgent          string `toml:"user_agent"`
	FetchMessagesLimit int    `toml:"fetch_messages_limit"`
	Mouse              bool   `toml:"mouse"`
	Timestamps         bool   `toml:"timestamps"`
}

type KeybindingsConfig struct {
	ToggleGuildsList         string `toml:"toggle_guilds_list"`
	ToggleChannelsTreeView   string `toml:"toggle_channels_tree_view"`
	ToggleMessagesTextView   string `toml:"toggle_messages_text_view"`
	ToggleMessageInputField  string `toml:"toggle_message_input_field"`
	ToggleMessageActionsList string `toml:"toggle_message_actions_list"`
	ToggleExternalEditor     string `toml:"toggle_external_editor"`

	SelectPreviousMessage string `toml:"select_previous_message"`
	SelectNextMessage     string `toml:"select_next_message"`
	SelectFirstMessage    string `toml:"select_first_message"`
	SelectLastMessage     string `toml:"select_last_message"`
}

type Config struct {
	Keybindings KeybindingsConfig `toml:"keybindings"`
	General     GeneralConfig     `toml:"general"`
}

func NewConfig() *Config {
	return &Config{
		General: GeneralConfig{
			UserAgent:          userAgent,
			FetchMessagesLimit: 50,
			Mouse:              true,
			Timestamps:         false,
		},
		Keybindings: KeybindingsConfig{
			ToggleGuildsList:         "Rune[g]",
			ToggleChannelsTreeView:   "Rune[c]",
			ToggleMessagesTextView:   "Rune[m]",
			ToggleMessageInputField:  "Rune[i]",
			ToggleMessageActionsList: "Rune[a]",
			ToggleExternalEditor:     "Ctrl-E",

			SelectPreviousMessage: "Up",
			SelectNextMessage:     "Down",
			SelectFirstMessage:    "Home",
			SelectLastMessage:     "End",
		},
	}
}

func LoadConfig() *Config {
	configPath, err := os.UserConfigDir()
	if err != nil {
		panic(err)
	}

	configPath += "/discordo/config.toml"
	// Create a directory as well as create all of the nested directories, recursively.
	err = os.MkdirAll(filepath.Dir(configPath), os.ModePerm)
	if err != nil {
		panic(err)
	}

	c := &Config{}
	// If the configuration file does not exist, create and write the default configuration to the file.
	if _, err = os.Stat(configPath); os.IsNotExist(err) {
		f, err := os.Create(configPath)
		if err != nil {
			panic(err)
		}
		defer f.Close()

		c = NewConfig()
		err = toml.NewEncoder(f).Encode(c)
		if err != nil {
			panic(err)
		}
	} else {
		_, err = toml.DecodeFile(configPath, c)
		if err != nil {
			panic(err)
		}
	}

	return c
}
