package config

import (
	charmKey "charm.land/bubbles/v2/key"
	myKey "github.com/ayn2op/discordo/pkg/key"
)

type Keybind struct {
	charmKey.Binding
}

func NewKeybind(key, desc string) Keybind {
	return Keybind{myKey.NewBinding(key, desc)}
}

func (k *Keybind) UnmarshalTOML(value any) error {
	var keys []string
	switch value := value.(type) {
	case string:
		keys = []string{value}
	case []string:
		keys = value
	default:
		return errInvalidType
	}
	k.SetKeys(keys...)
	return nil
}

type NavigationKeybinds struct {
	Up     Keybind `toml:"up"`
	Down   Keybind `toml:"down"`
	Top    Keybind `toml:"top"`
	Bottom Keybind `toml:"bottom"`
}

type ScrollKeybinds struct {
	ScrollUp     Keybind `toml:"scroll_up"`
	ScrollDown   Keybind `toml:"scroll_down"`
	ScrollTop    Keybind `toml:"scroll_top"`
	ScrollBottom Keybind `toml:"scroll_bottom"`
}

type SelectionKeybinds struct {
	SelectUp     Keybind `toml:"select_up"`
	SelectDown   Keybind `toml:"select_down"`
	SelectTop    Keybind `toml:"select_top"`
	SelectBottom Keybind `toml:"select_bottom"`
}

type TabsKeybinds struct {
	Previous Keybind `toml:"previous"`
	Next     Keybind `toml:"next"`
}

type InputKeybinds struct {
	OpenEditor Keybind `toml:"open_editor"`
}

type PickerKeybinds struct {
	NavigationKeybinds
	Toggle Keybind `toml:"toggle"`
	Cancel Keybind `toml:"cancel"`
	Select Keybind `toml:"select"`
}

type GuildsTreeKeybinds struct {
	NavigationKeybinds
	SelectCurrent Keybind `toml:"select_current"`
	YankID        Keybind `toml:"yank_id"`

	CollapseParentNode Keybind `toml:"collapse_parent_node"`
	MoveToParentNode   Keybind `toml:"move_to_parent_node"`
}

type MessagesListKeybinds struct {
	SelectionKeybinds
	ScrollKeybinds

	SelectReply  Keybind `toml:"select_reply"`
	Reply        Keybind `toml:"reply"`
	ReplyMention Keybind `toml:"reply_mention"`

	Cancel        Keybind `toml:"cancel"`
	Edit          Keybind `toml:"edit"`
	Delete        Keybind `toml:"delete"`
	DeleteConfirm Keybind `toml:"delete_confirm"`
	Open          Keybind `toml:"open"`

	YankContent Keybind `toml:"yank_content"`
	YankURL     Keybind `toml:"yank_url"`
	YankID      Keybind `toml:"yank_id"`
}

type MessageInputKeybinds struct {
	Paste       Keybind `toml:"paste"`
	Send        Keybind `toml:"send"`
	Cancel      Keybind `toml:"cancel"`
	TabComplete Keybind `toml:"tab_complete"`

	OpenEditor     Keybind `toml:"open_editor"`
	OpenFilePicker Keybind `toml:"open_file_picker"`
}

type MentionsListKeybinds struct {
	NavigationKeybinds
}

type Keybinds struct {
	ToggleGuildsTree  Keybind `toml:"toggle_guilds_tree"`
	FocusGuildsTree   Keybind `toml:"focus_guilds_tree"`
	FocusMessagesList Keybind `toml:"focus_messages_list"`
	FocusMessageInput Keybind `toml:"focus_message_input"`

	FocusPrevious Keybind `toml:"focus_previous"`
	FocusNext     Keybind `toml:"focus_next"`

	Tabs         TabsKeybinds         `toml:"tabs"`
	Input        InputKeybinds        `toml:"input"`
	Picker       PickerKeybinds       `toml:"picker"`
	GuildsTree   GuildsTreeKeybinds   `toml:"guilds_tree"`
	MessagesList MessagesListKeybinds `toml:"messages_list"`
	MessageInput MessageInputKeybinds `toml:"message_input"`
	MentionsList MentionsListKeybinds `toml:"mentions_list"`

	Logout  Keybind `toml:"logout"`
	Help    Keybind `toml:"help"`
	Suspend Keybind `toml:"suspend"`
	Quit    Keybind `toml:"quit"`
}

func defaultKeybinds() Keybinds {
	return Keybinds{
		FocusGuildsTree:   NewKeybind("ctrl+g", "guilds"),
		FocusMessagesList: NewKeybind("ctrl+t", "messages"),
		FocusMessageInput: NewKeybind("ctrl+i", "input"),

		FocusPrevious: NewKeybind("ctrl+h", "focus prev"),
		FocusNext:     NewKeybind("ctrl+l", "focus next"),

		Logout:  NewKeybind("ctrl+d", "logout"),
		Help:    NewKeybind("ctrl+.", "help"),
		Suspend: NewKeybind("ctrl+z", "suspend"),
		Quit:    NewKeybind("ctrl+c", "quit"),
	}
}
