package config

type (
	GuildsTreeKeys struct {
		Focus  string `yaml:"focus"`
		Toggle string `yaml:"toggle"`
	}

	MessagesTextKeys struct {
		Focus string `yaml:"focus"`

		ShowImage   string `yaml:"show_image"`
		CopyContent string `yaml:"copy_content"`

		Reply        string `yaml:"reply"`
		ReplyMention string `yaml:"reply_mention"`
		SelectReply  string `yaml:"select_reply"`

		SelectPrevious string `yaml:"select_previous"`
		SelectNext     string `yaml:"select_next"`
		SelectFirst    string `yaml:"select_first"`
		SelectLast     string `yaml:"select_last"`
	}

	MessageInputKeys struct {
		Focus string `yaml:"focus"`

		Send  string `yaml:"send"`
		Paste string `yaml:"paste"`

		LaunchEditor string `yaml:"launch_editor"`
	}

	BookmarkKeys struct {
		Slots []string `yaml:slots`
		FirstBookmark string `yaml:"first_bookmark"`
		PassBookmarks string `yaml:"pass_bookmarks"`
	}
)

type Keys struct {
	Cancel string `yaml:"cancel"`

	GuildsTree   GuildsTreeKeys   `yaml:"guilds_tree"`
	MessagesText MessagesTextKeys `yaml:"messages_text"`
	MessageInput MessageInputKeys `yaml:"message_input"`
	Bookmark BookmarkKeys     `yaml:"bookmark"`
}

func defKeys() Keys {
	return Keys{
		Cancel: "Esc",

		GuildsTree: GuildsTreeKeys{
			Focus:  "Alt+Rune[g]",
			Toggle: "Alt+Rune[b]",
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

		Bookmark: BookmarkKeys{
			Slots: []string{
				"Rune[1]",
				"Rune[2]",
				"Rune[3]",
				"Rune[4]",
				"Rune[5]",
				"Rune[6]",
				"Rune[7]",
				"Rune[8]",
				"Rune[9]",
				"Rune[0]",
			},
			FirstBookmark: "Rune[K]",
			PassBookmarks: "Rune[J]",
		},
	}
}
