package config

type NavigationKeybinds struct {
	Down   string `toml:"down"`
	Up     string `toml:"up"`
	Top    string `toml:"top"`
	Bottom string `toml:"bottom"`
}

type ScrollKeybinds struct {
	ScrollDown   string `toml:"scroll_down"`
	ScrollUp     string `toml:"scroll_up"`
	ScrollTop    string `toml:"scroll_top"`
	ScrollBottom string `toml:"scroll_bottom"`
}

type SelectionKeybinds struct {
	SelectDown   string `toml:"select_down"`
	SelectUp     string `toml:"select_up"`
	SelectTop    string `toml:"select_top"`
	SelectBottom string `toml:"select_bottom"`
}

type PickerKeybinds struct {
	NavigationKeybinds
	Toggle string `toml:"toggle"`
	Select string `toml:"select"`
	Cancel string `toml:"cancel"`
}

type GuildsTreeKeybinds struct {
	NavigationKeybinds
	SelectCurrent string `toml:"select_current"`
	YankID        string `toml:"yank_id"`
	Cancel        string `toml:"cancel"`

	CollapseParentNode string `toml:"collapse_parent_node"`
	MoveToParentNode   string `toml:"move_to_parent_node"`
}

type MessagesListKeybinds struct {
	SelectionKeybinds
	ScrollKeybinds

	SelectReply  string `toml:"select_reply"`
	ReplyMention string `toml:"reply_mention"`
	Reply        string `toml:"reply"`

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
	NavigationKeybinds
}

type HotkeysKeybinds struct {
	ShowAll string `toml:"show_all"`
}

type Keybinds struct {
	FocusGuildsTree   string `toml:"focus_guilds_tree"`
	FocusMessagesList string `toml:"focus_messages_list"`
	FocusMessageInput string `toml:"focus_message_input"`
	FocusPrevious     string `toml:"focus_previous"`
	FocusNext         string `toml:"focus_next"`
	ToggleGuildsTree  string `toml:"toggle_guilds_tree"`
	ToggleHotkeysBar  string `toml:"toggle_hotkeys_bar"`

	Picker       PickerKeybinds       `toml:"picker"`
	GuildsTree   GuildsTreeKeybinds   `toml:"guilds_tree"`
	MessagesList MessagesListKeybinds `toml:"messages_list"`
	MessageInput MessageInputKeybinds `toml:"message_input"`
	MentionsList MentionsListKeybinds `toml:"mentions_list"`
	Hotkeys      HotkeysKeybinds      `toml:"hotkeys_bar"`

	Logout string `toml:"logout"`
	Quit   string `toml:"quit"`
}
