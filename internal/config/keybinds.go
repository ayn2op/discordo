package config

type NavigationKeybinds struct {
	Up     string `toml:"up"`
	Down   string `toml:"down"`
	Top    string `toml:"top"`
	Bottom string `toml:"bottom"`
}

type ScrollKeybinds struct {
	ScrollUp     string `toml:"scroll_up"`
	ScrollDown   string `toml:"scroll_down"`
	ScrollTop    string `toml:"scroll_top"`
	ScrollBottom string `toml:"scroll_bottom"`
}

type SelectionKeybinds struct {
	SelectUp     string `toml:"select_up"`
	SelectDown   string `toml:"select_down"`
	SelectTop    string `toml:"select_top"`
	SelectBottom string `toml:"select_bottom"`
}

type PickerKeybinds struct {
	NavigationKeybinds
	Toggle string `toml:"toggle"`
	Cancel string `toml:"cancel"`
	Select string `toml:"select"`
}

type GuildsTreeKeybinds struct {
	NavigationKeybinds
	SelectCurrent string `toml:"select_current"`
	YankID        string `toml:"yank_id"`

	CollapseParentNode string `toml:"collapse_parent_node"`
	MoveToParentNode   string `toml:"move_to_parent_node"`
}

type MessagesListKeybinds struct {
	SelectionKeybinds
	ScrollKeybinds

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

type MessageInputKeybinds struct {
	Paste       string `toml:"paste"`
	Send        string `toml:"send"`
	Cancel      string `toml:"cancel"`
	TabComplete string `toml:"tab_complete"`

	OpenEditor     string `toml:"open_editor"`
	OpenFilePicker string `toml:"open_file_picker"`
}

type MentionsListKeybinds struct {
	SelectionKeybinds
}

type Keybinds struct {
	FocusGuildsTree   string `toml:"focus_guilds_tree"`
	FocusMessagesList string `toml:"focus_messages_list"`
	FocusMessageInput string `toml:"focus_message_input"`
	FocusPrevious     string `toml:"focus_previous"`
	FocusNext         string `toml:"focus_next"`
	ToggleGuildsTree  string `toml:"toggle_guilds_tree"`

	Picker       PickerKeybinds       `toml:"picker"`
	GuildsTree   GuildsTreeKeybinds   `toml:"guilds_tree"`
	MessagesList MessagesListKeybinds `toml:"messages_list"`
	MessageInput MessageInputKeybinds `toml:"message_input"`
	MentionsList MentionsListKeybinds `toml:"mentions_list"`

	Logout string `toml:"logout"`
	Quit   string `toml:"quit"`
}
