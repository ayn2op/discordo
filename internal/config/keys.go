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
		ToggleGuildsTree  string `toml:"toggle_guilds_tree" default:"Ctrl+B" description:"Toggle (show/hide) the guilds tree widget."`

		GuildsTree   GuildsTreeKeys   `toml:"guilds_tree"`
		MessagesList MessagesListKeys `toml:"messages_list"`
		MessageInput MessageInputKeys `toml:"message_input"`
		MentionsList MentionsListKeys `toml:"mentions_list"`

		Logout string `toml:"logout" default:"Ctrl+D" description:"Log out and remove the authentication token from keyring. Requires re-login upon restart."`
		Quit   string `toml:"quit" default:"Ctrl+C"`
	}

	GuildsTreeKeys struct {
		NavigationKeys
		SelectCurrent string `toml:"select_current" default:"Enter" description:"Select the currently highlighted text-based channel or expand a guild or channel."`
		YankID        string `toml:"yank_id" default:"Rune[i]" description:"Copy the ID of the currently highlighted node."`

		CollapseParentNode string `toml:"collapse_parent_node" default:"Rune[-]" description:"Collapse the currently highlighted guild or channel."`
		MoveToParentNode   string `toml:"move_to_parent_node" default:"Rune[p]" description:"Move to the parent guild or channel."`
	}

	MessagesListKeys struct {
		NavigationKeys
		SelectReply  string `toml:"select_reply" default:"Rune[s]" description:"Select the message reference (reply) of the selected channel."`
		Reply        string `toml:"reply" default:"Rune[R]" description:"Reply to the selected message."`
		ReplyMention string `toml:"reply_mention" default:"Rune[r]" description:"Reply (with mention) to the selected message."`

		Cancel        string `toml:"cancel" default:"Esc"`
		Edit          string `toml:"edit" default:"Rune[e]"`
		Delete        string `toml:"delete" default:"Rune[D]"`
		DeleteConfirm string `toml:"delete_confirm" default:"Rune[d]"`
		Open          string `toml:"open" default:"Rune[o]" description:"Open the selected message's attachments or hyperlinks in the message using the default browser application."`

		YankContent string `toml:"yank_content" default:"Rune[y]" description:"Copy/yank the selected message's content to clipboard."`
		YankURL     string `toml:"yank_url" default:"Rune[u]" description:"Copy/yank the selected message's URL to clipboard."`
		YankID      string `toml:"yank_id" default:"Rune[i]" description:"Copy/yank the selected message's ID to clipboard."`
	}

	MessageInputKeys struct {
		Paste       string `toml:"paste" default:"Ctrl+V" description:"Paste clipboard contents (supports both text and images) to message input widget."`
		Send        string `toml:"send" default:"Enter"`
		Cancel      string `toml:"cancel" default:"Esc"`
		TabComplete string `toml:"tab_complete" default:"@"`

		OpenEditor     string `toml:"open_editor" default:"Ctrl+E"`
		OpenFilePicker string `toml:"open_file_picker" default:"Ctrl+\\"`
	}

	MentionsListKeys struct {
		Up   string `toml:"up" default:"Ctrl+P"`
		Down string `toml:"down" default:"Ctrl+N"`
	}
)
