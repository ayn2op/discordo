package config

import (
	"encoding/json"
	"os"
)

type GeneralConfig struct {
	UserAgent          string `json:"userAgent"`
	FetchMessagesLimit int    `json:"fetchMessagesLimit"`
	Mouse              bool   `json:"mouse"`
	Notifications      bool   `json:"notifications"`
}

type KeybindingsConfig struct {
	FocusGuildsList        []string `json:"focusGuildsList"`
	FocusChannelsTreeView  []string `json:"focusChannelsTreeView"`
	FocusMessagesTextView  []string `json:"focusMessagesTextView"`
	FocusMessageInputField []string `json:"focusMessageInputField"`

	SelectPreviousMessage       []string `json:"selectPreviousMessage"`
	SelectNextMessage           []string `json:"selectNextMessage"`
	SelectFirstMessage          []string `json:"selectFirstMessage"`
	SelectLastMessage           []string `json:"selectLastMessage"`
	SelectMessageReference      []string `json:"selectMessageReference"`
	ReplySelectedMessage        []string `json:"replySelectedMessage"`
	MentionReplySelectedMessage []string `json:"mentionReplySelectedMessage"`
	CopySelectedMessage         []string `json:"copySelectedMessage"`
}

type Config struct {
	General     GeneralConfig     `json:"general"`
	Keybindings KeybindingsConfig `json:"keybindings"`
}

func New() *Config {
	return &Config{
		General: GeneralConfig{
			UserAgent:          "Mozilla/5.0 (X11; Linux x86_64; rv:95.0) Gecko/20100101 Firefox/95.0",
			FetchMessagesLimit: 50,
			Mouse:              true,
			Notifications:      true,
		},
		Keybindings: KeybindingsConfig{
			FocusGuildsList:        []string{"Alt+Rune[g]"},
			FocusChannelsTreeView:  []string{"Alt+Rune[t]"},
			FocusMessagesTextView:  []string{"Alt+Rune[m]"},
			FocusMessageInputField: []string{"Alt+Rune[i]"},

			SelectPreviousMessage:       []string{"Up"},
			SelectNextMessage:           []string{"Down"},
			SelectFirstMessage:          []string{"Home"},
			SelectLastMessage:           []string{"End"},
			SelectMessageReference:      []string{"Rune[m]"},
			ReplySelectedMessage:        []string{"Rune[r]"},
			MentionReplySelectedMessage: []string{"Rune[R]"},
			CopySelectedMessage:         []string{"Rune[c]"},
		},
	}
}

func (c *Config) Load() {
	configPath, err := os.UserConfigDir()
	if err != nil {
		panic(err)
	}

	configPath += "/discordo.json"
	f, err := os.OpenFile(configPath, os.O_CREATE|os.O_RDWR, os.ModePerm)
	if err != nil {
		panic(err)
	}

	fi, err := f.Stat()
	if err != nil {
		panic(err)
	}
	// If the size of the file is zero (the file is empty), write the default configuration to the file.
	if fi.Size() == 0 {
		e := json.NewEncoder(f)
		e.SetIndent("", "\t")

		c = New()
		err = e.Encode(c)
		if err != nil {
			panic(err)
		}
	} else {
		err = json.NewDecoder(f).Decode(c)
		if err != nil {
			panic(err)
		}
	}
}
