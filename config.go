package main

import (
	"os"

	"github.com/BurntSushi/toml"
	"github.com/rivo/tview"
)

const userAgent = "" +
	"Mozilla/5.0 (X11; Linux x86_64) " +
	"AppleWebKit/537.36 (KHTML, like Gecko) " +
	"Chrome/92.0.4515.131 Safari/537.36"

type keybindings struct {
	FocusChannelsTree      string `toml:"focus_channels_tree"`
	FocusMessagesView      string `toml:"focus_messages_view"`
	FocusMessageInputField string `toml:"focus_message_input_field"`

	SelectPreviousMessage       string `toml:"select_previous_message"`
	SelectNextMessage           string `toml:"select_next_message"`
	SelectFirstMessage          string `toml:"select_first_message"`
	SelectLastMessage           string `toml:"select_last_message"`
	ReplySelectedMessage        string `toml:"reply_selected_message"`
	MentionReplySelectedMessage string `toml:"mention_reply_selected_message"`
}

type theme struct {
	Border     string `toml:"border"`
	Title      string `toml:"title"`
	Background string `toml:"background"`
	Text       string `toml:"text"`
}

type borders struct {
	Horizontal  rune
	Vertical    rune
	TopLeft     rune
	TopRight    rune
	BottomLeft  rune
	BottomRight rune

	LeftT   rune
	RightT  rune
	TopT    rune
	BottomT rune
	Cross   rune

	HorizontalFocus  rune
	VerticalFocus    rune
	TopLeftFocus     rune
	TopRightFocus    rune
	BottomLeftFocus  rune
	BottomRightFocus rune
}

type config struct {
	Token            string `toml:"token"`
	UserAgent        string `toml:"user_agent"`
	Mouse            bool   `toml:"mouse"`
	Notifications    bool   `toml:"notifications"`
	GetMessagesLimit int    `toml:"get_messages_limit"`

	Theme       theme       `toml:"theme"`
	Keybindings keybindings `toml:"keybindings"`
	Borders     borders     `toml:"borders"`
}

func loadConfig() *config {
	configPath, err := os.UserConfigDir()
	if err != nil {
		panic(err)
	}

	configPath += "/discordo.toml"

	var c config
	if _, err = os.Stat(configPath); os.IsNotExist(err) {
		f, err := os.Create(configPath)
		if err != nil {
			panic(err)
		}
		defer f.Close()

		c.Mouse = true
		c.Notifications = true
		c.UserAgent = userAgent
		c.GetMessagesLimit = 50
		c.Theme = theme{
			Border:     "white",
			Title:      "cyan",
			Background: "black",
			Text:       "white",
		}
		c.Keybindings = keybindings{
			FocusChannelsTree:      "Alt+Left",
			FocusMessagesView:      "Alt+Right",
			FocusMessageInputField: "Alt+Down",

			SelectPreviousMessage:       "Up",
			SelectNextMessage:           "Down",
			SelectFirstMessage:          "Home",
			SelectLastMessage:           "End",
			ReplySelectedMessage:        "Rune[r]",
			MentionReplySelectedMessage: "Rune[R]",
		}
		c.Borders = tview.Borders

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

	return &c
}
