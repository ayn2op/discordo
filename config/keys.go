package config

type (
	MessagesTextKeys struct {
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
		Send  string `yaml:"send"`
		Paste string `yaml:"paste"`

		LaunchEditor string `yaml:"launch_editor"`
	}
)

type Keys struct {
	Cancel string `yaml:"cancel"`

	MessagesText MessagesTextKeys `yaml:"messages_text"`
	MessageInput MessageInputKeys `yaml:"message_input"`
}

func newKeys() Keys {
	return Keys{
		Cancel: "Esc",

		MessagesText: MessagesTextKeys{
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
			Send: "Enter",

			Paste:        "Ctrl+V",
			LaunchEditor: "Ctrl+E",
		},
	}
}
