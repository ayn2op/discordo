package main

import (
	"os"

	"github.com/BurntSushi/toml"
)

var conf *config

type keybindingsChannelsTree struct {
	Focus string `toml:"focus"`
}

type keybindingsMessagesTextView struct {
	Focus          string `toml:"focus"`
	SelectPrevious string `toml:"select_previous"`
	SelectNext     string `toml:"select_next"`
	SelectFirst    string `toml:"select_first"`
	SelectLast     string `toml:"select_last"`
	Reply          string `toml:"reply"`
	ReplyMention   string `toml:"reply_mention"`
}

type keybindingsMessageInputField struct {
	Focus string `toml:"focus"`
}

type keybindings struct {
	ChannelsTree      keybindingsChannelsTree      `toml:"channels_tree"`
	MessagesTextView  keybindingsMessagesTextView  `toml:"messages_textview"`
	MessageInputField keybindingsMessageInputField `toml:"message_inputfield"`
}

type themeBackground struct {
	// Main background color for primitives.
	Primitive string `toml:"primitive"`
	// Background color for contrasting elements.
	Contrast string `toml:"contrast"`
	// Background color for even more contrasting elements.
	MoreContrast string `toml:"more_contrast"`
}

type themeText struct {
	// Primary text.
	Primary string `toml:"primary"`
	// Secondary text (e.g. labels).
	Secondary string `toml:"secondary"`
	// Tertiary text (e.g. subtitles, notes).
	Tertiary string `toml:"tertiary"`
	// Text on primary-colored backgrounds.
	Inverse string `toml:"inverse"`
	// Secondary text on ContrastBackgroundColor-colored backgrounds.
	ContrastSecondary string `toml:"contrast_secondary"`
}

type theme struct {
	// Box borders.
	Border string `toml:"border"`
	// Box titles.
	Title string `toml:"title"`
	// Graphics.
	Graphics string `toml:"graphics"`

	Background themeBackground `toml:"background"`
	Text       themeText       `toml:"text"`
}

type config struct {
	Token            string      `toml:"token"`
	Mouse            bool        `toml:"mouse"`
	Notifications    bool        `toml:"notifications"`
	UserAgent        string      `toml:"user_agent"`
	GetMessagesLimit int         `toml:"get_messages_limit"`
	Theme            theme       `toml:"theme"`
	Keybindings      keybindings `toml:"keybindings"`
}

func loadConfig() *config {
	u, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}

	configPath := u + "/.config/discordo/config.toml"
	if _, err = os.Stat(configPath); os.IsNotExist(err) {
		err = os.MkdirAll(u+"/.config/discordo", 0700)
		if err != nil {
			panic(err)
		}

		f, err := os.Create(configPath)
		if err != nil {
			panic(err)
		}
		defer f.Close()

		c := config{
			Mouse:         true,
			Notifications: true,
			UserAgent: "" +
				"Mozilla/5.0 (X11; Linux x86_64) " +
				"AppleWebKit/537.36 (KHTML, like Gecko) " +
				"Chrome/92.0.4515.131 Safari/537.36",
			GetMessagesLimit: 50,
			Theme: theme{
				Border:   "white",
				Title:    "white",
				Graphics: "white",
				Background: themeBackground{
					Primitive:    "black",
					Contrast:     "blue",
					MoreContrast: "green",
				},
				Text: themeText{
					Primary:           "white",
					Secondary:         "yellow",
					Tertiary:          "green",
					Inverse:           "blue",
					ContrastSecondary: "darkcyan",
				},
			},
			Keybindings: keybindings{
				ChannelsTree: keybindingsChannelsTree{
					Focus: "Alt+Rune[1]",
				},
				MessagesTextView: keybindingsMessagesTextView{
					Focus:          "Alt+Rune[2]",
					SelectPrevious: "Up",
					SelectNext:     "Down",
					SelectFirst:    "Home",
					SelectLast:     "End",
					Reply:          "Rune[r]",
					ReplyMention:   "Rune[R]",
				},
				MessageInputField: keybindingsMessageInputField{
					Focus: "Alt+Rune[3]",
				},
			},
		}

		err = toml.NewEncoder(f).Encode(c)
		if err != nil {
			panic(err)
		}

		return &c
	}

	var c config
	_, err = toml.DecodeFile(configPath, &c)
	if err != nil {
		panic(err)
	}

	return &c
}
