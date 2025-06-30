package config

type (
	NavigationKeys struct {
		SelectPrevious string `toml:"select_previous"`
		SelectNext     string `toml:"select_next"`
		SelectFirst    string `toml:"select_first"`
		SelectLast     string `toml:"select_last"`
	}

	Keys struct {
		FocusGuildsTree   string `toml:"focus_guilds_tree"`
		FocusMessagesList string `toml:"focus_messages_list"`
		FocusMessageInput string `toml:"focus_message_input"`
		ToggleGuildsTree  string `toml:"toggle_guilds_tree"`

		GuildsTree   GuildsTreeKeys   `toml:"guilds_tree"`
		MessagesList MessagesListKeys `toml:"messages_list"`
		MessageInput MessageInputKeys `toml:"message_input"`
		MentionsList MentionsListKeys `toml:"mentions_list"`

		Logout string `toml:"logout"`
		Quit   string `toml:"quit"`
	}

	GuildsTreeKeys struct {
		NavigationKeys
		SelectCurrent string `toml:"select_current"`
		YankID        string `toml:"yank_id"`

		CollapseParentNode string `toml:"collapse_parent_node"`
		MoveToParentNode   string `toml:"move_to_parent_node"`

		NextUnread     string `toml:"next_unread"`
		PreviousUnread string `toml:"previous_unread"`
	}

	MessagesListKeys struct {
		NavigationKeys
		SelectReply  string `toml:"select_reply"`
		Reply        string `toml:"reply"`
		ReplyMention string `toml:"reply_mention"`

		Cancel string `toml:"cancel"`
		Delete string `toml:"delete"`
		Open   string `toml:"open"`

		YankContent string `toml:"yank_content"`
		YankURL     string `toml:"yank_url"`
		YankID      string `toml:"yank_id"`
	}

	MessageInputKeys struct {
		Send        string `toml:"send"`
		Editor      string `toml:"editor"`
		Cancel      string `toml:"cancel"`
		TabComplete string `toml:"tab_complete"`
	}

	MentionsListKeys struct {
		Up   string `toml:"up"`
		Down string `toml:"down"`
	}
)
