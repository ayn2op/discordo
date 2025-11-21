package config

type (
	NavigationKeys struct {
		SelectPrevious string `toml:"select_previous" default:"Rune[k]"`
		SelectNext     string `toml:"select_next" default:"Rune[j]"`
		SelectFirst    string `toml:"select_first" default:"Rune[g]"`
		SelectLast     string `toml:"select_last" default:"Rune[G]"`
	}

	Keys struct {
		FocusGuildsTree   string `toml:"focus_guilds_tree" default:"Ctrl+G"`
		FocusMessagesList string `toml:"focus_messages_list" default:"Ctrl+T"`
		FocusMessageInput string `toml:"focus_message_input" default:"Ctrl+Space"`
		FocusPrevious     string `toml:"focus_previous" default:"Ctrl+H"`
		FocusNext         string `toml:"focus_next" default:"Ctrl+L"`
		ToggleGuildsTree  string `toml:"toggle_guilds_tree" default:"Ctrl+B"`

		GuildsTree   GuildsTreeKeys   `toml:"guilds_tree"`
		MessagesList MessagesListKeys `toml:"messages_list"`
		MessageInput MessageInputKeys `toml:"message_input"`
		MentionsList MentionsListKeys `toml:"mentions_list"`

		Logout string `toml:"logout" default:"Ctrl+D"`
		Quit   string `toml:"quit" default:"Ctrl+C"`
	}

	GuildsTreeKeys struct {
		NavigationKeys
		SelectCurrent string `toml:"select_current" default:"Enter"`

		YankID             string `toml:"yank_id" default:"Rune[i]"`
		CollapseParentNode string `toml:"collapse_parent_node"`
		MoveToParentNode   string `toml:"move_to_parent_node"`
	}

	MessagesListKeys struct {
		NavigationKeys
		SelectReply string `toml:"select_reply" default:"Rune[s]"`

		Reply        string `toml:"reply" default:"Rune[R]"`
		ReplyMention string `toml:"reply_mention" default:"Rune[r]"`

		Cancel        string `toml:"cancel" default:"Esc"`
		Edit          string `toml:"edit" default:"Rune[e]"`
		Delete        string `toml:"delete" default:"Rune[D]"`
		DeleteConfirm string `toml:"delete_confirm" default:"Rune[d]"`
		Open          string `toml:"open" default:"Rune[o]"`

		YankContent string `toml:"yank_content" default:"Rune[y]"`
		YankURL     string `toml:"yank_url" default:"Rune[u]"`
		YankID      string `toml:"yank_id" default:"Rune[i]"`
	}

	MessageInputKeys struct {
		Paste       string `toml:"paste" default:"Ctrl+V"`
		Send        string `toml:"send" default:"Enter"`
		Cancel      string `toml:"cancel" default:"Esc"`
		TabComplete string `toml:"tab_complete" default:"Tab"`

		OpenEditor     string `toml:"open_editor" default:"Ctrl+E"`
		OpenFilePicker string `toml:"open_file_picker" default:"Ctrl+\\"`
	}

	MentionsListKeys struct {
		Up   string `toml:"up" default:"Ctrl+P"`
		Down string `toml:"down" default:"Ctrl+N"`
	}
)
