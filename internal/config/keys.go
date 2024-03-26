package config

type (
	Keys struct {
		Normal NormalModeKeys `toml:"normal"`
		Insert InsertModeKeys `toml:"insert"`
	}

	NormalModeKeys struct {
		InsertMode        string `toml:"insert_mode"`
		FocusGuildsTree   string `toml:"focus_guilds_tree"`
		FocusMessagesText string `toml:"focus_messages_text"`
		ToggleGuildsTree  string `toml:"toggle_guild_tree"`

		GuildsTree   GuildsTreeNormalModeKeys   `toml:"guilds_tree"`
		MessagesText MessagesTextNormalModeKeys `toml:"messages_text"`
	}

	GuildsTreeNormalModeKeys struct {
		SelectCurrent  string `toml:"select_current"`
		SelectPrevious string `toml:"select_previous"`
		SelectNext     string `toml:"select_next"`
		SelectFirst    string `toml:"select_first"`
		SelectLast     string `toml:"select_last"`
	}

	MessagesTextNormalModeKeys struct {
		SelectPrevious string `toml:"select_previous"`
		SelectNext     string `toml:"select_next"`
		SelectFirst    string `toml:"select_first"`
		SelectLast     string `toml:"select_last"`
		SelectReply    string `toml:"select_reply"`

		Reply        string `toml:"reply"`
		ReplyMention string `toml:"reply_mention"`

		Delete string `toml:"delete"`
		Yank   string `toml:"yank"`
		Open   string `toml:"open"`
	}

	InsertModeKeys struct {
		NormalMode string `toml:"normal_mode"`

		MessageInput MessageInputInsertModeKeys `toml:"message_input"`
	}

	MessageInputInsertModeKeys struct {
		Send   string `toml:"send"`
		Editor string `toml:"editor"`
	}
)

func defaultKeys() Keys {
	return Keys{
		Normal: NormalModeKeys{
			InsertMode: "Rune[i]",

			FocusGuildsTree:   "Ctrl+G",
			FocusMessagesText: "Ctrl+T",
			ToggleGuildsTree:  "Ctrl+B",

			GuildsTree: GuildsTreeNormalModeKeys{
				SelectCurrent:  "Enter",
				SelectPrevious: "Rune[k]",
				SelectNext:     "Rune[j]",
				SelectFirst:    "Rune[g]",
				SelectLast:     "Rune[G]",
			},
			MessagesText: MessagesTextNormalModeKeys{
				SelectPrevious: "Rune[k]",
				SelectNext:     "Rune[j]",
				SelectFirst:    "Rune[g]",
				SelectLast:     "Rune[G]",
				SelectReply:    "Rune[s]",

				Reply:        "Rune[r]",
				ReplyMention: "Rune[R]",

				Delete: "Rune[d]",
				Yank:   "Rune[y]",
				Open:   "Rune[o]",
			},
		},
		Insert: InsertModeKeys{
			NormalMode: "Esc",
			MessageInput: MessageInputInsertModeKeys{
				Send:   "Enter",
				Editor: "Ctrl+E",
			},
		},
	}
}
