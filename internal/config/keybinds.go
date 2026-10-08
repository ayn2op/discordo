package config

import (
	"encoding/json"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/ayn2op/tview/keybind"
)

type Keybind struct {
	keybind.Keybind
}

var _ toml.Unmarshaler = (*Keybind)(nil)

func (k *Keybind) UnmarshalTOML(value any) error {
	var keys []string
	switch value := value.(type) {
	case string:
		keys = []string{value}
	case []any:
		for _, key := range value {
			if key, ok := key.(string); ok {
				keys = append(keys, key)
			}
		}
	default:
		return nil
	}
	kb := keybind.New(keys...)
	// Keep displayed help key aligned with configured key(s).
	help := k.Help()
	if keys := kb.Keys(); len(keys) > 0 {
		help.Key = keys[0]
	}
	k.Keybind = kb.WithHelp(help.Key, help.Desc)
	return nil
}

// MarshalTOML writes the keys as UnmarshalTOML reads them: one key as a string, an unbound keybind as an empty one, and several keys as a list.
func (k Keybind) MarshalTOML() ([]byte, error) {
	if keys := k.Keys(); len(keys) > 1 {
		return json.Marshal(keys)
	}
	return json.Marshal(strings.Join(k.Keys(), ""))
}

// key builds a Keybind bound to key, or unbound if key is empty, with desc as its help description.
func key(key, desc string) Keybind {
	k := Keybind{keybind.New().WithHelp("", desc)}
	k.UnmarshalTOML(key)
	return k
}

type ScrollKeybinds struct {
	// Scroll the messages list without changing the selection.
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

type TreeKeybinds struct {
	SelectionKeybinds
	// Select the highlighted text-based channel, or expand or collapse the highlighted node.
	SelectCurrent Keybind `toml:"select_current"`
	YankID        Keybind `toml:"yank_id"`

	CollapseAll        Keybind `toml:"collapse_all"`
	CollapseParentNode Keybind `toml:"collapse_parent_node"`
	MoveToParentNode   Keybind `toml:"move_to_parent_node"`
}

type GuildsTreeKeybinds struct {
	TreeKeybinds
	// Highlight the previous or next unread guild or channel, wrapping around.
	SelectPreviousUnread Keybind `toml:"select_previous_unread"`
	SelectNextUnread     Keybind `toml:"select_next_unread"`
	// Join the highlighted voice channel to listen, or leave it.
	ToggleVoice Keybind `toml:"toggle_voice"`
}

type MessagesListKeybinds struct {
	SelectionKeybinds
	ScrollKeybinds

	// Select the message that the selected message replies to.
	SelectReply Keybind `toml:"select_reply"`
	// Reply to the selected message.
	Reply Keybind `toml:"reply"`
	// Reply (with mention) to the selected message.
	ReplyMention Keybind `toml:"reply_mention"`

	Cancel Keybind `toml:"cancel"`
	Edit   Keybind `toml:"edit"`

	// Delete without confirmation. Unbound by default.
	Delete        Keybind `toml:"delete"`
	DeleteConfirm Keybind `toml:"delete_confirm"`

	// Open attachments with their associated application. MIME types outside the
	// allowlist require confirmation. Hyperlinks still use the browser.
	Open Keybind `toml:"open"`
	// Open attachments or hyperlinks in the browser.
	OpenInBrowser Keybind `toml:"open_in_browser"`
	// Download one of the selected message's attachments.
	Download Keybind `toml:"download"`

	// Yank (copy) the selected message's content/url/id.
	YankContent Keybind `toml:"yank_content"`
	YankURL     Keybind `toml:"yank_url"`
	YankID      Keybind `toml:"yank_id"`
}

type ComposerKeybinds struct {
	// Edit your newest editable message loaded in the current channel, with no age limit.
	// Only when the composer is empty, with no reply, attachments, or edit in progress.
	EditLast Keybind `toml:"edit_last"`
	// Paste from clipboard (supports both text and images).
	Paste Keybind `toml:"paste"`
	Send  Keybind `toml:"send"`
	// Insert a new line at the cursor.
	Newline Keybind `toml:"newline"`
	// Remove existing text or cancel reply.
	Cancel Keybind `toml:"cancel"`
	// Complete usernames when mentioning.
	TabComplete Keybind `toml:"tab_complete"`
	// Toggle whether a reply mentions its author.
	ToggleReplyMention Keybind `toml:"toggle_reply_mention"`
	Undo               Keybind `toml:"undo"`

	OpenEditor     Keybind `toml:"open_editor"`
	OpenFilePicker Keybind `toml:"open_file_picker"`
}

type MentionsListKeybinds struct {
	SelectionKeybinds
}

type Keybinds struct {
	// Hide/show the guilds tree.
	ToggleGuildsTree Keybind `toml:"toggle_guilds_tree"`
	// Hide/show the members tree.
	ToggleMembersTree    Keybind `toml:"toggle_members_tree"`
	ToggleChannelsPicker Keybind `toml:"toggle_channels_picker"`
	// Start or stop sending what the microphone captures to the voice channel joined.
	ToggleMute Keybind `toml:"toggle_mute"`
	// Stop or start playing what is said in the voice channel joined. Stopping also mutes.
	ToggleDeafen Keybind `toml:"toggle_deafen"`
	// Hide/show keybind help.
	ToggleHelp Keybind `toml:"toggle_help"`
	// Switch between compact and full keybind help.
	ToggleFullHelp Keybind `toml:"toggle_full_help"`
	Suspend        Keybind `toml:"suspend"`

	FocusGuildsTree   Keybind `toml:"focus_guilds_tree"`
	FocusMembersTree  Keybind `toml:"focus_members_tree"`
	FocusMessagesList Keybind `toml:"focus_messages_list"`
	FocusComposer     Keybind `toml:"focus_composer"`

	// Cycle focus between the widgets.
	FocusPrevious Keybind `toml:"focus_previous"`
	FocusNext     Keybind `toml:"focus_next"`

	Picker PickerKeybinds `toml:"picker"`
	// Only while focusing on the guilds tree.
	GuildsTree GuildsTreeKeybinds `toml:"guilds_tree"`
	// Only while focusing on the members tree.
	MembersTree TreeKeybinds `toml:"members_tree"`
	// Only while focusing on sent messages.
	MessagesList MessagesListKeybinds `toml:"messages_list"`
	// Only while typing a message.
	Composer ComposerKeybinds `toml:"composer"`
	// Only while the mention suggestions are shown.
	MentionsList MentionsListKeybinds `toml:"mentions_list"`

	// Log out and remove the authentication token from keyring.
	// Requires re-login upon restart.
	Logout Keybind `toml:"logout"`
	Quit   Keybind `toml:"quit"`
}

func selectionKeybinds(up, down, top, bottom string) SelectionKeybinds {
	return SelectionKeybinds{
		SelectUp:     key(up, "up"),
		SelectDown:   key(down, "down"),
		SelectTop:    key(top, "top"),
		SelectBottom: key(bottom, "btm"),
	}
}

func defaultKeybinds() Keybinds {
	vi := selectionKeybinds("k", "j", "g", "G")
	emacs := selectionKeybinds("ctrl+p", "ctrl+n", "home", "end")
	tree := TreeKeybinds{
		SelectionKeybinds: vi,
		SelectCurrent:     key("enter", "select"),
		YankID:            key("i", "copy id"),

		CollapseAll:        key("_", "collapse all"),
		CollapseParentNode: key("-", "collapse parent"),
		MoveToParentNode:   key("p", "parent"),
	}
	return Keybinds{
		ToggleGuildsTree:     key("alt+g", "toggle guilds"),
		ToggleMembersTree:    key("alt+m", "toggle members"),
		ToggleChannelsPicker: key("ctrl+k", "channels picker"),
		ToggleMute:           key("alt+v", "toggle mute"),
		ToggleDeafen:         key("alt+d", "toggle deafen"),
		ToggleHelp:           key("alt+.", "toggle help"),
		ToggleFullHelp:       key("ctrl+.", "full help"),
		Suspend:              key("ctrl+z", "suspend"),

		FocusGuildsTree:   key("ctrl+g", "guilds"),
		FocusMembersTree:  key("ctrl+y", "members"),
		FocusMessagesList: key("ctrl+t", "messages"),
		FocusComposer:     key("ctrl+i", "composer"),

		FocusPrevious: key("ctrl+h", "focus prev"),
		FocusNext:     key("ctrl+l", "focus next"),

		Picker: PickerKeybinds{
			SelectionKeybinds: emacs,
			Select:            key("enter", "sel"),
			Cancel:            key("esc", "cancel"),
		},
		GuildsTree: GuildsTreeKeybinds{
			TreeKeybinds:         tree,
			SelectPreviousUnread: key("K", "prev unread"),
			SelectNextUnread:     key("J", "next unread"),
			ToggleVoice:          key("v", "join voice"),
		},
		MembersTree: tree,
		MessagesList: MessagesListKeybinds{
			SelectionKeybinds: vi,
			ScrollKeybinds: ScrollKeybinds{
				ScrollUp:     key("K", "scr up"),
				ScrollDown:   key("J", "scr down"),
				ScrollTop:    key("home", "scr top"),
				ScrollBottom: key("end", "scr btm"),
			},

			SelectReply:  key("s", "sel reply"),
			Reply:        key("R", "reply"),
			ReplyMention: key("r", "@reply"),

			Cancel: key("esc", "cancel"),
			Edit:   key("e", "edit"),

			Delete:        key("", "force delete"),
			DeleteConfirm: key("d", "delete"),

			Open:          key("o", "open"),
			OpenInBrowser: key("O", "open browser"),
			Download:      key("D", "download"),

			YankContent: key("y", "copy text"),
			YankURL:     key("u", "copy url"),
			YankID:      key("i", "copy id"),
		},
		Composer: ComposerKeybinds{
			EditLast:           key("up", "edit last"),
			Paste:              key("ctrl+v", "paste"),
			Send:               key("enter", "send"),
			Newline:            key("shift+enter", "nl"),
			Cancel:             key("esc", "cancel"),
			TabComplete:        key("tab", "complete"),
			ToggleReplyMention: key("shift+tab", "toggle @reply"),
			Undo:               key("ctrl+u", "undo"),

			OpenEditor:     key("ctrl+e", "editor"),
			OpenFilePicker: key("ctrl+\\", "attach"),
		},
		MentionsList: MentionsListKeybinds{SelectionKeybinds: emacs},

		Logout: key("ctrl+d", "logout"),
		Quit:   key("ctrl+c", "quit"),
	}
}
