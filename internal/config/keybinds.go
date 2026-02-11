package config

type NavigationKeybinds struct {
	Down   string `toml:"down" join:"next"`
	Up     string `toml:"up" name:"next/prev" hot:"true"`
	Top    string `toml:"top" join:"next"`
	Bottom string `toml:"bottom" name:"first/last" hot:"true"`
}

type ScrollKeybinds struct {
	ScrollDown   string `toml:"scroll_down" join:"next"`
	ScrollUp     string `toml:"scroll_up" name:"scroll" hot:"true"`
	ScrollTop    string `toml:"scroll_top" join:"next"`
	ScrollBottom string `toml:"scroll_bottom" name"top/bot"`
}

type SelectionKeybinds struct {
	SelectDown   string `toml:"select_down" join:"next"`
	SelectUp     string `toml:"select_up" name:"next/prev" hot:"true"`
	SelectTop    string `toml:"select_top" join:"next"`
	SelectBottom string `toml:"select_bottom" name:"first/last" hot:"true"`
}

type PickerKeybinds struct {
	NavigationKeybinds
	Toggle string `toml:"toggle"`
	Select string `toml:"select" name:"select" hot:"true"`
	Cancel string `toml:"cancel" name:"cancel" hot:"true"`
}

type GuildsTreeKeybinds struct {
	NavigationKeybinds
	SelectCurrent string `toml:"select_current"`
	YankID        string `toml:"yank_id" name:"yank_id"`
	Cancel        string `toml:"cancel" name:"cancel" hot:"true"`

	CollapseParentNode string `toml:"collapse_parent_node" name:"collapse_parent"`
	MoveToParentNode   string `toml:"move_to_parent_node" name:"goto_parent"`
}

type MessagesListKeybinds struct {
	SelectionKeybinds
	ScrollKeybinds

	SelectReply  string `toml:"select_reply" name:"goto_reply" hot:"true"`
	ReplyMention string `toml:"reply_mention" join:"next"`
	Reply        string `toml:"reply" name:"@/reply" hot:"true"`

	Cancel        string `toml:"cancel" name:"cancel" hot:"true"`
	Edit          string `toml:"edit" name:"edit" hot:"true"`
	Delete        string `toml:"delete" name:"delete_force"`
	DeleteConfirm string `toml:"delete_confirm" name:"delete" hot:"true"`
	Open          string `toml:"open" name:"open" hot:"true"`

	YankContent string `toml:"yank_content" name:"yank_content"`
	YankURL     string `toml:"yank_url" name:"yank_url"`
	YankID      string `toml:"yank_id" name:"yank_id"`
}

type MessageInputKeybinds struct {
	Paste       string `toml:"paste" name:"paste" hot:"true"`
	Send        string `toml:"send" name:"send" hot:"true"`
	Cancel      string `toml:"cancel" name:"cancel" hot:"true"`
	TabComplete string `toml:"tab_complete"`

	OpenEditor     string `toml:"open_editor" name:"editor" hot:"true"`
	OpenFilePicker string `toml:"open_file_picker" name:"attach" hot:"true"`
}

type MentionsListKeybinds struct {
	NavigationKeybinds
}

type HotkeysKeybinds struct {
	ShowAll string `toml:"show_all"`
}

type Keybinds struct {
	FocusGuildsTree      string `toml:"focus_guilds_tree"`
	FocusMessagesList    string `toml:"focus_messages_list"`
	FocusMessageInput    string `toml:"focus_message_input"`
	FocusPrevious        string `toml:"focus_previous"`
	FocusNext            string `toml:"focus_next"`
	ToggleGuildsTree     string `toml:"toggle_guilds_tree"`

	Picker       PickerKeybinds       `toml:"picker"`
	GuildsTree   GuildsTreeKeybinds   `toml:"guilds_tree"`
	MessagesList MessagesListKeybinds `toml:"messages_list"`
	MessageInput MessageInputKeybinds `toml:"message_input"`
	MentionsList MentionsListKeybinds `toml:"mentions_list"`
	Hotkeys      HotkeysKeybinds      `toml:"hotkeys_bar"`

	Logout string `toml:"logout"`
	Quit   string `toml:"quit"`
}
