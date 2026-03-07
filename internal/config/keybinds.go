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

func newKeybind(key, desc string) Keybind {
	return Keybind{
		Keybind: keybind.NewKeybind(
			keybind.WithKeys(key),
			keybind.WithHelp(key, desc),
		),
	}
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

type PickerKeybinds struct {
	NavigationKeybinds
	Select Keybind `toml:"select"`
	Cancel Keybind `toml:"cancel"`
}

type GuildsTreeKeybinds struct {
	NavigationKeybinds
	SelectCurrent Keybind `toml:"select_current"`
	YankID        Keybind `toml:"yank_id"`

	CollapseParentNode Keybind `toml:"collapse_parent_node"`
	MoveToParentNode   Keybind `toml:"move_to_parent_node"`
	ShowVoiceUsers     Keybind `toml:"show_voice_channel_users"`
	HideVoiceUsers     Keybind `toml:"hide_voice_channel_users"`
	ShowAllVoiceUsers  Keybind `toml:"show_all_voice_channel_users"`
	HideAllVoiceUsers  Keybind `toml:"hide_all_voice_channel_users"`
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
	Undo        Keybind `toml:"undo"`

	OpenEditor     Keybind `toml:"open_editor"`
	OpenFilePicker Keybind `toml:"open_file_picker"`
}

type MentionsListKeybinds struct {
	NavigationKeybinds
}

type VoiceKeybinds struct {
	LeaveVoice   Keybind `toml:"leave_voice"`
	ToggleMute   Keybind `toml:"toggle_mute"`
	ToggleDeafen Keybind `toml:"toggle_deafen"`
}

type Keybinds struct {
	ToggleGuildsTree     Keybind `toml:"toggle_guilds_tree"`
	ToggleChannelsPicker Keybind `toml:"toggle_channels_picker"`
	ToggleHelp           Keybind `toml:"toggle_help"`
	Suspend              Keybind `toml:"suspend"`

	FocusGuildsTree   Keybind `toml:"focus_guilds_tree"`
	FocusMessagesList Keybind `toml:"focus_messages_list"`
	FocusMessageInput Keybind `toml:"focus_message_input"`

	FocusPrevious Keybind `toml:"focus_previous"`
	FocusNext     Keybind `toml:"focus_next"`

	Picker       PickerKeybinds       `toml:"picker"`
	GuildsTree   GuildsTreeKeybinds   `toml:"guilds_tree"`
	MessagesList MessagesListKeybinds `toml:"messages_list"`
	MessageInput MessageInputKeybinds `toml:"message_input"`
	MentionsList MentionsListKeybinds `toml:"mentions_list"`
	Voice        VoiceKeybinds        `toml:"voice"`

	Logout Keybind `toml:"logout"`
	Quit   Keybind `toml:"quit"`
}

func defaultPickerKeybinds() PickerKeybinds {
	return PickerKeybinds{
		NavigationKeybinds: NavigationKeybinds{
			Up:     newKeybind("ctrl+p", "up"),
			Down:   newKeybind("ctrl+n", "down"),
			Top:    newKeybind("home", "top"),
			Bottom: newKeybind("end", "bottom"),
		},
		Cancel: newKeybind("esc", "cancel"),
		Select: newKeybind("enter", "sel"),
	}
}

func defaultNavigationKeybinds() NavigationKeybinds {
	return NavigationKeybinds{
		Up:     newKeybind("k", "up"),
		Down:   newKeybind("j", "down"),
		Top:    newKeybind("g", "top"),
		Bottom: newKeybind("G", "bottom"),
	}
}

func defaultGuildsTreeKeybinds() GuildsTreeKeybinds {
	return GuildsTreeKeybinds{
		NavigationKeybinds: defaultNavigationKeybinds(),
		SelectCurrent:      newKeybind("enter", "sel"),
		YankID:             newKeybind("i", "copy id"),
		CollapseParentNode: newKeybind("-", "collapse"),
		MoveToParentNode:   newKeybind("p", "parent"),
		ShowVoiceUsers:     newKeybind("v", "show voice"),
		HideVoiceUsers:     newKeybind("V", "hide voice"),
		ShowAllVoiceUsers:  newKeybind("a", "show all"),
		HideAllVoiceUsers:  newKeybind("A", "hide all"),
	}
}

func defaultMessagesListKeybinds() MessagesListKeybinds {
	return MessagesListKeybinds{
		SelectionKeybinds: SelectionKeybinds{
			SelectUp:     newKeybind("k", "up"),
			SelectDown:   newKeybind("j", "down"),
			SelectTop:    newKeybind("g", "top"),
			SelectBottom: newKeybind("G", "bottom"),
		},
		ScrollKeybinds: ScrollKeybinds{
			ScrollUp:     newKeybind("K", "scroll up"),
			ScrollDown:   newKeybind("J", "scroll down"),
			ScrollTop:    newKeybind("home", "scroll top"),
			ScrollBottom: newKeybind("end", "scroll bottom"),
		},
		SelectReply:  newKeybind("s", "sel reply"),
		Reply:        newKeybind("R", "reply"),
		ReplyMention: newKeybind("r", "@reply"),
		Cancel:       newKeybind("esc", "cancel"),
		Edit:         newKeybind("e", "edit"),
		Delete:       newKeybind("D", "force delete"),
		DeleteConfirm: newKeybind(
			"d",
			"delete",
		),
		Open:        newKeybind("o", "open"),
		YankContent: newKeybind("y", "copy text"),
		YankURL:     newKeybind("u", "copy url"),
		YankID:      newKeybind("i", "copy id"),
	}
}

func defaultMessageInputKeybinds() MessageInputKeybinds {
	return MessageInputKeybinds{
		Paste:          newKeybind("ctrl+v", "paste"),
		Send:           newKeybind("enter", "send"),
		Cancel:         newKeybind("esc", "cancel"),
		TabComplete:    newKeybind("tab", "complete"),
		Undo:           newKeybind("ctrl+u", "undo"),
		OpenEditor:     newKeybind("ctrl+e", "editor"),
		OpenFilePicker: newKeybind("ctrl+\\", "attach"),
	}
}

func defaultMentionsListKeybinds() MentionsListKeybinds {
	return MentionsListKeybinds{
		NavigationKeybinds: NavigationKeybinds{
			Up:     newKeybind("ctrl+p", "up"),
			Down:   newKeybind("ctrl+n", "down"),
			Top:    newKeybind("home", "top"),
			Bottom: newKeybind("end", "bottom"),
		},
	}
}

func defaultVoiceKeybinds() VoiceKeybinds {
	return VoiceKeybinds{
		LeaveVoice:   newKeybind("q", "leave voice"),
		ToggleMute:   newKeybind("m", "mute"),
		ToggleDeafen: newKeybind("M", "deafen"),
	}
}

func defaultKeybinds() Keybinds {
	return Keybinds{
		ToggleGuildsTree:     newKeybind("ctrl+b", "toggle guilds"),
		ToggleChannelsPicker: newKeybind("ctrl+k", "channels picker"),
		ToggleHelp:           newKeybind("ctrl+.", "help"),
		Suspend:              newKeybind("ctrl+z", "suspend"),

		FocusGuildsTree:   newKeybind("ctrl+g", "guilds"),
		FocusMessagesList: newKeybind("ctrl+t", "messages"),
		FocusMessageInput: newKeybind("ctrl+i", "input"),

		FocusPrevious: newKeybind("ctrl+h", "focus prev"),
		FocusNext:     newKeybind("ctrl+l", "focus next"),

		Logout: newKeybind("ctrl+d", "logout"),
		Quit:   newKeybind("ctrl+c", "quit"),

		Picker:       defaultPickerKeybinds(),
		GuildsTree:   defaultGuildsTreeKeybinds(),
		MessagesList: defaultMessagesListKeybinds(),
		MessageInput: defaultMessageInputKeybinds(),
		MentionsList: defaultMentionsListKeybinds(),
		Voice:        defaultVoiceKeybinds(),
	}
}
