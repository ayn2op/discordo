package config

type (
	SelectionKeybinds struct {
		SelectPrevious string `toml:"select_previous"`
		SelectNext     string `toml:"select_next"`
		SelectFirst    string `toml:"select_first"`
		SelectLast     string `toml:"select_last"`
	}

	Keybinds struct {
		FocusGuildsTree   string `toml:"focus_guilds_tree"`
		FocusMessagesList string `toml:"focus_messages_list"`
		FocusMessageInput string `toml:"focus_message_input"`
		FocusPrevious     string `toml:"focus_previous"`
		FocusNext         string `toml:"focus_next"`
		ToggleGuildsTree  string `toml:"toggle_guilds_tree"`

		GuildsTree   GuildsTreeKeybinds   `toml:"guilds_tree"`
		MessagesList MessagesListKeybinds `toml:"messages_list"`
		MessageInput MessageInputKeybinds `toml:"message_input"`
		MentionsList MentionsListKeybinds `toml:"mentions_list"`

		Logout string `toml:"logout"`
		Quit   string `toml:"quit"`
	}

	GuildsTreeKeybinds struct {
		SelectionKeybinds
		SelectCurrent string `toml:"select_current"`
		YankID        string `toml:"yank_id"`

		CollapseParentNode string `toml:"collapse_parent_node"`
		MoveToParentNode   string `toml:"move_to_parent_node"`
	}

	MessagesListKeybinds struct {
		SelectionKeybinds
		ScrollUp     string `toml:"scroll_up"`
		ScrollDown   string `toml:"scroll_down"`
		ScrollTop    string `toml:"scroll_top"`
		ScrollBottom string `toml:"scroll_bottom"`

		SelectReply  string `toml:"select_reply"`
		Reply        string `toml:"reply"`
		ReplyMention string `toml:"reply_mention"`

		Cancel        string `toml:"cancel"`
		Edit          string `toml:"edit"`
		Delete        string `toml:"delete"`
		DeleteConfirm string `toml:"delete_confirm"`
		Open          string `toml:"open"`

		YankContent string `toml:"yank_content"`
		YankURL     string `toml:"yank_url"`
		YankID      string `toml:"yank_id"`
	}

	MessageInputKeybinds struct {
		Paste       string `toml:"paste"`
		Send        string `toml:"send"`
		Cancel      string `toml:"cancel"`
		TabComplete string `toml:"tab_complete"`

		OpenEditor     string `toml:"open_editor"`
		OpenFilePicker string `toml:"open_file_picker"`
	}

	MentionsListKeybinds struct {
		Up   string `toml:"up"`
		Down string `toml:"down"`
	}
)
