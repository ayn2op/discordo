package util

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
	FocusChannelsTree      []string `toml:"focus_channels_tree"`
	FocusMessagesView      []string `toml:"focus_messages_view"`
	FocusMessageInputField []string `toml:"focus_message_input_field"`

	SelectPreviousMessage       []string `toml:"select_previous_message"`
	SelectNextMessage           []string `toml:"select_next_message"`
	SelectFirstMessage          []string `toml:"select_first_message"`
	SelectLastMessage           []string `toml:"select_last_message"`
	SelectMessageReference      []string `toml:"select_message_reference"`
	ReplySelectedMessage        []string `toml:"reply_selected_message"`
	MentionReplySelectedMessage []string `toml:"mention_reply_selected_message"`
	CopySelectedMessage         []string `toml:"copy_selected_message"`
}

type theme struct {
	Background string `toml:"background"`

	Border   string `toml:"border"`
	Title    string `toml:"title"`
	Graphics string `toml:"graphics"`
	Text     string `toml:"text"`
}

type borders struct {
	Horizontal  rune
	Vertical    rune
	TopLeft     rune
	TopRight    rune
	BottomLeft  rune
	BottomRight rune

	HorizontalFocus  rune
	VerticalFocus    rune
	TopLeftFocus     rune
	TopRightFocus    rune
	BottomLeftFocus  rune
	BottomRightFocus rune
}

type Config struct {
	Token            string `toml:"token"`
	UserAgent        string `toml:"user_agent"`
	Mouse            bool   `toml:"mouse"`
	Notifications    bool   `toml:"notifications"`
	GetMessagesLimit int    `toml:"get_messages_limit"`

	Theme       theme       `toml:"theme"`
	Keybindings keybindings `toml:"keybindings"`
	Borders     borders     `toml:"borders"`
}

// LoadConfig loads the configuration file, if the configuration file exists or creates a new one if not, and returns it.
func LoadConfig() *Config {
	configPath, err := os.UserConfigDir()
	if err != nil {
		panic(err)
	}

	configPath += "/discordo.toml"

	var c Config
	if _, err = os.Stat(configPath); os.IsNotExist(err) {
		f, err := os.Create(configPath)
		if err != nil {
			panic(err)
		}
		defer f.Close()

		c = Config{
			Mouse:            true,
			Notifications:    true,
			UserAgent:        userAgent,
			GetMessagesLimit: 50,
			Borders: borders{
				Horizontal:  0,
				Vertical:    0,
				TopLeft:     0,
				TopRight:    0,
				BottomLeft:  0,
				BottomRight: 0,

				HorizontalFocus:  tview.BoxDrawingsLightHorizontal,
				VerticalFocus:    tview.BoxDrawingsLightVertical,
				TopLeftFocus:     tview.BoxDrawingsLightDownAndRight,
				TopRightFocus:    tview.BoxDrawingsLightDownAndLeft,
				BottomLeftFocus:  tview.BoxDrawingsLightUpAndRight,
				BottomRightFocus: tview.BoxDrawingsLightUpAndLeft,
			},
			Theme: theme{
				Background: "black",

				Border:   "white",
				Title:    "white",
				Graphics: "white",
				Text:     "white",
			},
			Keybindings: keybindings{
				FocusChannelsTree:      []string{"Alt+Left"},
				FocusMessagesView:      []string{"Alt+Right"},
				FocusMessageInputField: []string{"Alt+Down"},

				SelectPreviousMessage:       []string{"Up"},
				SelectNextMessage:           []string{"Down"},
				SelectFirstMessage:          []string{"Home"},
				SelectLastMessage:           []string{"End"},
				ReplySelectedMessage:        []string{"Rune[r]"},
				MentionReplySelectedMessage: []string{"Rune[R]"},
				CopySelectedMessage:         []string{"Rune[c]"},
				SelectMessageReference:      []string{"Rune[m]"},
			},
		}
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
