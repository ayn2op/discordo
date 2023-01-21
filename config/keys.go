package config

type (
	GuildsTreeKeys struct {
		Focus string `yaml:"focus"`
	}

	MessagesTextKeys struct {
		Focus string `yaml:"focus"`

		CopyContent string `yaml:"copy_content"`

		Reply        string `yaml:"reply"`
		ReplyMention string `yaml:"reply_mention"`
		SelectReply  string `yaml:"select_reply"`

		SelectPrevious string `yaml:"select_previous"`
		SelectNext     string `yaml:"select_next"`
		SelectFirst    string `yaml:"select_first"`
		SelectLast     string `yaml:"select_last"`

		ShowImage string `yaml:"show_image"`
	}

	MessageInputKeys struct {
		Focus string `yaml:"focus"`

		Send  string `yaml:"send"`
		Paste string `yaml:"paste"`

		LaunchEditor string `yaml:"launch_editor"`
	}
)

type Keys struct {
	Cancel string `yaml:"cancel"`

	GuildsTree   GuildsTreeKeys   `yaml:"guilds_tree"`
	MessagesText MessagesTextKeys `yaml:"messages_text"`
	MessageInput MessageInputKeys `yaml:"message_input"`
}

func newKeys() Keys {
	return Keys{
		Cancel: "Esc",

		GuildsTree: GuildsTreeKeys{
			Focus: "Alt+Rune[g]",
		},

		MessagesText: MessagesTextKeys{
			Focus: "Alt+Rune[m]",

			ShowImage:   "Rune[i]",
			CopyContent: "Rune[c]",

			Reply:        "Rune[r]",
			ReplyMention: "Rune[R]",
			SelectReply:  "Rune[s]",

			SelectPrevious: "Up",
			SelectNext:     "Down",
			SelectFirst:    "Home",
			SelectLast:     "End",
		},

		MessageInput: MessageInputKeys{
			Focus: "Alt+Rune[i]",

			Send: "Enter",

			Paste:        "Ctrl+V",
			LaunchEditor: "Ctrl+E",
		},
	}
}
