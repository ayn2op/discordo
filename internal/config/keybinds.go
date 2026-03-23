package config

import (
	"github.com/BurntSushi/toml"
	"github.com/ayn2op/tview/keybind"
)

type Keybind struct {
	keybind.Keybind
}

var _ toml.Unmarshaler = (*Keybind)(nil)

func (k *Keybind) UnmarshalTOML(value any) error {
	switch value := value.(type) {
	case string:
		k.SetKeys(value)
	case []any:
		keys := make([]string, 0, len(value))
		for _, key := range value {
			if key, ok := key.(string); ok {
				keys = append(keys, key)
			}
		}
		k.SetKeys(keys...)
	}
	// Keep displayed help key aligned with configured key(s).
	if keys := k.Keys(); len(keys) > 0 {
		k.SetHelp(keys[0], k.Help().Desc)
	}
	return nil
}

type NavigationKeybinds struct {
	Up     Keybind `toml:"up" default:"k" help:"up"`
	Down   Keybind `toml:"down" default:"j" help:"down"`
	Top    Keybind `toml:"top" default:"g" help:"top"`
	Bottom Keybind `toml:"bottom" default:"G" help:"bottom"`
}

type ScrollKeybinds struct {
	ScrollUp     Keybind `toml:"scroll_up" default:"K" help:"scroll up"`
	ScrollDown   Keybind `toml:"scroll_down" default:"J" help:"scroll down"`
	ScrollTop    Keybind `toml:"scroll_top" default:"home" help:"scroll top"`
	ScrollBottom Keybind `toml:"scroll_bottom" default:"end" help:"scroll bottom"`
}

type SelectionKeybinds struct {
	SelectUp     Keybind `toml:"select_up" default:"k" help:"up"`
	SelectDown   Keybind `toml:"select_down" default:"j" help:"down"`
	SelectTop    Keybind `toml:"select_top" default:"g" help:"top"`
	SelectBottom Keybind `toml:"select_bottom" default:"G" help:"bottom"`
}

type PickerKeybinds struct {
	Up     Keybind `toml:"up" default:"ctrl+p" help:"up"`
	Down   Keybind `toml:"down" default:"ctrl+n" help:"down"`
	Top    Keybind `toml:"top" default:"home" help:"top"`
	Bottom Keybind `toml:"bottom" default:"end" help:"bottom"`
	Select Keybind `toml:"select" default:"enter" help:"sel"`
	Cancel Keybind `toml:"cancel" default:"esc" help:"cancel"`
}

type GuildsTreeKeybinds struct {
	NavigationKeybinds

	SelectCurrent Keybind `toml:"select_current" default:"enter" help:"sel"`
	YankID        Keybind `toml:"yank_id" default:"i" help:"copy id"`

	CollapseParentNode Keybind `toml:"collapse_parent_node" default:"-" help:"collapse"`
	MoveToParentNode   Keybind `toml:"move_to_parent_node" default:"p" help:"parent"`
}

type MessagesListKeybinds struct {
	SelectionKeybinds
	ScrollKeybinds

	SelectReply  Keybind `toml:"select_reply" default:"s" help:"sel reply"`
	Reply        Keybind `toml:"reply" default:"R" help:"reply"`
	ReplyMention Keybind `toml:"reply_mention" default:"r" help:"@reply"`

	Cancel        Keybind `toml:"cancel" default:"esc" help:"cancel"`
	Edit          Keybind `toml:"edit" default:"e" help:"edit"`
	Delete        Keybind `toml:"delete" default:"D" help:"force delete"`
	DeleteConfirm Keybind `toml:"delete_confirm" default:"d" help:"delete"`
	Open          Keybind `toml:"open" default:"o" help:"open"`

	YankContent Keybind `toml:"yank_content" default:"y" help:"copy text"`
	YankURL     Keybind `toml:"yank_url" default:"u" help:"copy url"`
	YankID      Keybind `toml:"yank_id" default:"i" help:"copy id"`
}

type MessageInputKeybinds struct {
	Paste       Keybind `toml:"paste" default:"ctrl+v" help:"paste"`
	Send        Keybind `toml:"send" default:"enter" help:"send"`
	Cancel      Keybind `toml:"cancel" default:"esc" help:"cancel"`
	TabComplete Keybind `toml:"tab_complete" default:"tab" help:"complete"`
	Undo        Keybind `toml:"undo" default:"ctrl+u" help:"undo"`

	OpenEditor     Keybind `toml:"open_editor" default:"ctrl+e" help:"editor"`
	OpenFilePicker Keybind `toml:"open_file_picker" default:"ctrl+\\" help:"attach"`
}

type MentionsListKeybinds struct {
	Up     Keybind `toml:"up" default:"ctrl+p" help:"up"`
	Down   Keybind `toml:"down" default:"ctrl+n" help:"down"`
	Top    Keybind `toml:"top" default:"home" help:"top"`
	Bottom Keybind `toml:"bottom" default:"end" help:"bottom"`
}

type Keybinds struct {
	ToggleGuildsTree     Keybind `toml:"toggle_guilds_tree" default:"ctrl+b" help:"toggle guilds"`
	ToggleChannelsPicker Keybind `toml:"toggle_channels_picker" default:"ctrl+k" help:"channels picker"`
	ToggleHelp           Keybind `toml:"toggle_help" default:"ctrl+." help:"help"`
	Suspend              Keybind `toml:"suspend" default:"ctrl+z" help:"suspend"`

	FocusGuildsTree   Keybind `toml:"focus_guilds_tree" default:"ctrl+g" help:"guilds"`
	FocusMessagesList Keybind `toml:"focus_messages_list" default:"ctrl+t" help:"messages"`
	FocusMessageInput Keybind `toml:"focus_message_input" default:"ctrl+i" help:"input"`

	FocusPrevious Keybind `toml:"focus_previous" default:"ctrl+h" help:"focus prev"`
	FocusNext     Keybind `toml:"focus_next" default:"ctrl+l" help:"focus next"`

	Picker       PickerKeybinds       `toml:"picker"`
	GuildsTree   GuildsTreeKeybinds   `toml:"guilds_tree"`
	MessagesList MessagesListKeybinds `toml:"messages_list"`
	MessageInput MessageInputKeybinds `toml:"message_input"`
	MentionsList MentionsListKeybinds `toml:"mentions_list"`

	Logout Keybind `toml:"logout" default:"ctrl+d" help:"logout"`
	Quit   Keybind `toml:"quit" default:"ctrl+c" help:"quit"`
}
