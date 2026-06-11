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

// desc builds a Keybind with only a help description; keys come from config.toml.
func desc(s string) Keybind {
	return Keybind{
		Keybind: keybind.NewKeybind(keybind.WithHelp("", s)),
	}
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
	SelectionKeybinds
	Select Keybind `toml:"select"`
	Cancel Keybind `toml:"cancel"`
}

type GuildsTreeKeybinds struct {
	SelectionKeybinds
	SelectCurrent Keybind `toml:"select_current"`
	YankID        Keybind `toml:"yank_id"`

	CollapseAll        Keybind `toml:"collapse_all"`
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

type ComposerKeybinds struct {
	Paste       Keybind `toml:"paste"`
	Send        Keybind `toml:"send"`
	Newline     Keybind `toml:"newline"`
	Cancel      Keybind `toml:"cancel"`
	TabComplete Keybind `toml:"tab_complete"`
	Undo        Keybind `toml:"undo"`

	OpenEditor     Keybind `toml:"open_editor"`
	OpenFilePicker Keybind `toml:"open_file_picker"`
}

type MentionsListKeybinds struct {
	SelectionKeybinds
}

type Keybinds struct {
	ToggleGuildsTree     Keybind `toml:"toggle_guilds_tree"`
	ToggleChannelsPicker Keybind `toml:"toggle_channels_picker"`
	ToggleHelp           Keybind `toml:"toggle_help"`
	Suspend              Keybind `toml:"suspend"`

	FocusGuildsTree   Keybind `toml:"focus_guilds_tree"`
	FocusMessagesList Keybind `toml:"focus_messages_list"`
	FocusComposer     Keybind `toml:"focus_composer"`

	FocusPrevious Keybind `toml:"focus_previous"`
	FocusNext     Keybind `toml:"focus_next"`

	Picker       PickerKeybinds       `toml:"picker"`
	GuildsTree   GuildsTreeKeybinds   `toml:"guilds_tree"`
	MessagesList MessagesListKeybinds `toml:"messages_list"`
	Composer     ComposerKeybinds     `toml:"composer"`
	MentionsList MentionsListKeybinds `toml:"mentions_list"`

	Logout Keybind `toml:"logout"`
	Quit   Keybind `toml:"quit"`
}

func defaultSelectionKeybinds() SelectionKeybinds {
	return SelectionKeybinds{
		SelectUp:     desc("up"),
		SelectDown:   desc("down"),
		SelectTop:    desc("top"),
		SelectBottom: desc("btm"),
	}
}

func defaultPickerKeybinds() PickerKeybinds {
	return PickerKeybinds{
		SelectionKeybinds: defaultSelectionKeybinds(),
		Cancel:            desc("cancel"),
		Select:            desc("sel"),
	}
}

func defaultGuildsTreeKeybinds() GuildsTreeKeybinds {
	return GuildsTreeKeybinds{
		SelectionKeybinds: defaultSelectionKeybinds(),
		SelectCurrent:     desc("select"),
		YankID:            desc("copy id"),

		CollapseAll:        desc("collapse all"),
		CollapseParentNode: desc("collapse parent"),
		MoveToParentNode:   desc("parent"),
	}
}

func defaultMessagesListKeybinds() MessagesListKeybinds {
	return MessagesListKeybinds{
		SelectionKeybinds: defaultSelectionKeybinds(),
		ScrollKeybinds: ScrollKeybinds{
			ScrollUp:     desc("scr up"),
			ScrollDown:   desc("scr down"),
			ScrollTop:    desc("scr top"),
			ScrollBottom: desc("scr btm"),
		},
		SelectReply:   desc("sel reply"),
		Reply:         desc("reply"),
		ReplyMention:  desc("@reply"),
		Cancel:        desc("cancel"),
		Edit:          desc("edit"),
		Delete:        desc("force delete"),
		DeleteConfirm: desc("delete"),
		Open:          desc("open"),
		YankContent:   desc("copy text"),
		YankURL:       desc("copy url"),
		YankID:        desc("copy id"),
	}
}

func defaultComposerKeybinds() ComposerKeybinds {
	return ComposerKeybinds{
		Paste:          desc("paste"),
		Send:           desc("send"),
		Newline:        desc("nl"),
		Cancel:         desc("cancel"),
		TabComplete:    desc("complete"),
		Undo:           desc("undo"),
		OpenEditor:     desc("editor"),
		OpenFilePicker: desc("attach"),
	}
}

func defaultMentionsListKeybinds() MentionsListKeybinds {
	return MentionsListKeybinds{
		SelectionKeybinds: defaultSelectionKeybinds(),
	}
}

func defaultKeybinds() Keybinds {
	return Keybinds{
		ToggleGuildsTree:     desc("toggle guilds"),
		ToggleChannelsPicker: desc("channels picker"),
		ToggleHelp:           desc("help"),
		Suspend:              desc("suspend"),

		FocusGuildsTree:   desc("guilds"),
		FocusMessagesList: desc("messages"),
		FocusComposer:     desc("composer"),

		FocusPrevious: desc("focus prev"),
		FocusNext:     desc("focus next"),

		Logout: desc("logout"),
		Quit:   desc("quit"),

		Picker:       defaultPickerKeybinds(),
		GuildsTree:   defaultGuildsTreeKeybinds(),
		MessagesList: defaultMessagesListKeybinds(),
		Composer:     defaultComposerKeybinds(),
		MentionsList: defaultMentionsListKeybinds(),
	}
}
