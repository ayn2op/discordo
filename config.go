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

type keybindingsChannelsTree struct {
	Focus string
}

type keybindingsMessagesView struct {
	Focus          string
	SelectPrevious string
	SelectNext     string
	SelectFirst    string
	SelectLast     string
	Reply          string
	ReplyMention   string
}

type keybindingsMessageInputField struct {
	Focus string
}

type keybindings struct {
	ChannelsTree      keybindingsChannelsTree
	MessagesView      keybindingsMessagesView
	MessageInputField keybindingsMessageInputField
}

type themeBackground struct {
	// Main background color for primitives.
	Primitive string
	// Background color for contrasting elements.
	Contrast string
	// Background color for even more contrasting elements.
	MoreContrast string
}

type themeText struct {
	// Primary text.
	Primary string
	// Secondary text (e.g. labels).
	Secondary string
	// Tertiary text (e.g. subtitles, notes).
	Tertiary string
	// Text on primary-colored backgrounds.
	Inverse string
	// Secondary text on ContrastBackgroundColor-colored backgrounds.
	ContrastSecondary string
}

type theme struct {
	// Box borders.
	Border string
	// Box titles.
	Title string
	// Graphics.
	Graphics string

	Background themeBackground
	Text       themeText
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
	Token            string
	UserAgent        string
	Mouse            bool
	Notifications    bool
	GetMessagesLimit int
	Theme            theme
	Keybindings      keybindings
	Borders          borders
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
		}
		c.Keybindings = keybindings{
			ChannelsTree: keybindingsChannelsTree{
				Focus: "Alt+Rune[1]",
			},
			MessagesView: keybindingsMessagesView{
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
