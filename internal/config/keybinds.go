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

// key builds a Keybind with desc as its help description, bound to keys, the first of which is shown in help. It is unbound if that is empty.
func key(desc string, keys ...string) Keybind {
	return Keybind{keybind.New(keys...).WithHelp(keys[0], desc)}
}

type ScrollKeybinds struct {
	// Scroll without changing the selection.
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
	ScrollKeybinds
	// Scroll sideways to read names that are cut off.
	ScrollLeft  Keybind `toml:"scroll_left"`
	ScrollRight Keybind `toml:"scroll_right"`

	SelectionKeybinds
	// Select the highlighted text-based channel, or expand or collapse the highlighted node.
	SelectCurrent Keybind `toml:"select_current"`

	YankID Keybind `toml:"yank_id"`

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
	// Play, pause or resume the selected voice message.
	Play Keybind `toml:"play"`
	// Seek the voice message that is playing or paused.
	SeekBackward Keybind `toml:"seek_backward"`
	SeekForward  Keybind `toml:"seek_forward"`

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
	LeaveVoice   Keybind `toml:"leave_voice"`
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

func selectionKeybinds(up, down []string, top, bottom string) SelectionKeybinds {
	return SelectionKeybinds{
		SelectUp:     key("up", up...),
		SelectDown:   key("down", down...),
		SelectTop:    key("top", top),
		SelectBottom: key("btm", bottom),
	}
}

func scrollKeybinds(up, down, top, bottom string) ScrollKeybinds {
	return ScrollKeybinds{
		ScrollUp:     key("scr up", up),
		ScrollDown:   key("scr down", down),
		ScrollTop:    key("scr top", top),
		ScrollBottom: key("scr btm", bottom),
	}
}

func defaultKeybinds() Keybinds {
	vi := selectionKeybinds([]string{"k", "up"}, []string{"j", "down"}, "g", "G")
	emacs := selectionKeybinds([]string{"ctrl+p"}, []string{"ctrl+n"}, "home", "end")
	tree := TreeKeybinds{
		SelectionKeybinds: vi,
		ScrollKeybinds:    scrollKeybinds("pgup", "pgdn", "home", "end"),
		ScrollLeft:        key("scr left", "h"),
		ScrollRight:       key("scr right", "l"),
		SelectCurrent:     key("select", "enter"),
		YankID:            key("copy id", "i"),

		CollapseAll:        key("collapse all", "_"),
		CollapseParentNode: key("collapse parent", "-"),
		MoveToParentNode:   key("parent", "p"),
	}
	return Keybinds{
		ToggleGuildsTree:     key("toggle guilds", "alt+g"),
		ToggleMembersTree:    key("toggle members", "alt+m"),
		ToggleChannelsPicker: key("channels picker", "ctrl+k"),
		ToggleMute:           key("toggle mute", "alt+v"),
		ToggleDeafen:         key("toggle deafen", "alt+d"),
		LeaveVoice:           key("leave voice", "alt+l"),
		ToggleHelp:           key("toggle help", "alt+."),
		ToggleFullHelp:       key("full help", "ctrl+."),
		Suspend:              key("suspend", "ctrl+z"),

		FocusGuildsTree:   key("guilds", "ctrl+g"),
		FocusMembersTree:  key("members", "ctrl+y"),
		FocusMessagesList: key("messages", "ctrl+t"),
		FocusComposer:     key("composer", "ctrl+i"),

		FocusPrevious: key("focus prev", "ctrl+h"),
		FocusNext:     key("focus next", "ctrl+l"),

		Picker: PickerKeybinds{
			SelectionKeybinds: emacs,
			Select:            key("sel", "enter"),
			Cancel:            key("cancel", "esc"),
		},
		GuildsTree: GuildsTreeKeybinds{
			TreeKeybinds:         tree,
			SelectPreviousUnread: key("prev unread", "K"),
			SelectNextUnread:     key("next unread", "J"),
			ToggleVoice:          key("join voice", "v"),
		},
		MembersTree: tree,
		MessagesList: MessagesListKeybinds{
			SelectionKeybinds: vi,
			ScrollKeybinds:    scrollKeybinds("K", "J", "home", "end"),

			SelectReply:  key("sel reply", "s"),
			Reply:        key("reply", "R"),
			ReplyMention: key("@reply", "r"),

			Cancel: key("cancel", "esc"),
			Edit:   key("edit", "e"),

			Delete:        key("force delete", ""),
			DeleteConfirm: key("delete", "d"),

			Open:          key("open", "o"),
			OpenInBrowser: key("open browser", "O"),
			Download:      key("download", "D"),
			Play:          key("play", "p"),
			SeekBackward:  key("seek back", "h", "left"),
			SeekForward:   key("seek forward", "l", "right"),

			YankContent: key("copy text", "y"),
			YankURL:     key("copy url", "u"),
			YankID:      key("copy id", "i"),
		},
		Composer: ComposerKeybinds{
			EditLast:           key("edit last", "up"),
			Paste:              key("paste", "ctrl+v"),
			Send:               key("send", "enter"),
			Newline:            key("nl", "shift+enter"),
			Cancel:             key("cancel", "esc"),
			TabComplete:        key("complete", "tab"),
			ToggleReplyMention: key("toggle @reply", "shift+tab"),
			Undo:               key("undo", "ctrl+u"),

			OpenEditor:     key("editor", "ctrl+e"),
			OpenFilePicker: key("attach", "ctrl+\\"),
		},
		MentionsList: MentionsListKeybinds{SelectionKeybinds: emacs},

		Logout: key("logout", "ctrl+d"),
		Quit:   key("quit", "ctrl+c"),
	}
}
