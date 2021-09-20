package main

import (
	"encoding/json"
	"os"

	"github.com/rivo/tview"
)

type keybindings struct {
	GuildsTreeViewFocus string

	MessagesTextViewFocus                string
	MessagesTextViewSelectPrevious       string
	MessagesTextViewSelectNext           string
	MessagesTextViewSelectFirst          string
	MessagesTextViewSelectLast           string
	MessagesTextViewReplySelected        string
	MessagesTextViewMentionReplySelected string

	MessageInputFieldFocus string
}

type config struct {
	Token            string
	Mouse            bool
	Notifications    bool
	UserAgent        string
	GetMessagesLimit int
	Theme            tview.Theme
	Keybindings      keybindings
}

func loadConfig() *config {
	u, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}

	configPath := u + "/.config/discordo/config.json"
	if _, err = os.Stat(configPath); os.IsNotExist(err) {
		f, err := os.Create(configPath)
		if err != nil {
			panic(err)
		}

		c := config{
			Mouse:         true,
			Notifications: true,
			UserAgent: "" +
				"Mozilla/5.0 (X11; Linux x86_64) " +
				"AppleWebKit/537.36 (KHTML, like Gecko) " +
				"Chrome/92.0.4515.131 Safari/537.36",
			GetMessagesLimit: 50,
			Theme:            tview.Styles,
			Keybindings: keybindings{
				GuildsTreeViewFocus: "Alt+Rune[1]",

				MessagesTextViewFocus:                "Alt+Rune[2]",
				MessagesTextViewSelectPrevious:       "Up",
				MessagesTextViewSelectNext:           "Down",
				MessagesTextViewSelectFirst:          "Home",
				MessagesTextViewSelectLast:           "End",
				MessagesTextViewReplySelected:        "r",
				MessagesTextViewMentionReplySelected: "R",

				MessageInputFieldFocus: "Alt+Rune[3]",
			},
		}
		d, err := json.MarshalIndent(c, "", "\t")
		if err != nil {
			panic(err)
		}

		_, err = f.Write(d)
		if err != nil {
			panic(err)
		}

		f.Sync()
	}

	d, err := os.ReadFile(configPath)
	if err != nil {
		panic(err)
	}

	var c config
	if err = json.Unmarshal(d, &c); err != nil {
		panic(err)
	}

	return &c
}
