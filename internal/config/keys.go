package config

type (
	Keys struct {
		FocusGuildsTree   string `toml:"focus_guilds_tree"`
		FocusMessagesText string `toml:"focus_messages_text"`
		FocusMessageInput string `toml:"focus_message_input"`
		ToggleGuildsTree  string `toml:"toggle_guild_tree"`

		SelectPrevious string `toml:"select_previous"`
		SelectNext     string `toml:"select_next"`
		SelectFirst    string `toml:"select_first"`
		SelectLast     string `toml:"select_last"`

		GuildsTree   GuildsTreeKeys   `toml:"guilds_tree"`
		MessagesText MessagesTextKeys `toml:"messages_text"`
		MessageInput MessageInputKeys `toml:"message_input"`
	}

	GuildsTreeKeys struct {
		SelectCurrent string `toml:"select_current"`
	}

	MessagesTextKeys struct {
		SelectReply  string `toml:"select_reply"`
		Reply        string `toml:"reply"`
		ReplyMention string `toml:"reply_mention"`

		Delete string `toml:"delete"`
		Yank   string `toml:"yank"`
		Open   string `toml:"open"`
	}

	MessageInputKeys struct {
		Send   string `toml:"send"`
		Editor string `toml:"editor"`
		Cancel string `toml:"cancel"`
	}
)

func defaultKeys() Keys {
	return Keys{
		FocusGuildsTree:   "Ctrl+G",
		FocusMessagesText: "Ctrl+T",
		FocusMessageInput: "Ctrl+P",
		ToggleGuildsTree:  "Ctrl+B",

		SelectPrevious: "Rune[k]",
		SelectNext:     "Rune[j]",
		SelectFirst:    "Rune[g]",
		SelectLast:     "Rune[G]",

		GuildsTree: GuildsTreeKeys{
			SelectCurrent: "Enter",
		},

		MessagesText: MessagesTextKeys{
			SelectReply: "Rune[s]",

			Reply:        "Rune[r]",
			ReplyMention: "Rune[R]",

			Delete: "Rune[d]",
			Yank:   "Rune[y]",
			Open:   "Rune[o]",
		},

		MessageInput: MessageInputKeys{
			Send:   "Enter",
			Editor: "Ctrl+E",
			Cancel: "Esc",
		},
	}
}
