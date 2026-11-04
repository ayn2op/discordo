package config

import (
	"github.com/ayn2op/tview/help"
	"github.com/ayn2op/tview/keybind"
)

var _ help.KeyMap = PickerKeybinds{}

func (k PickerKeybinds) ShortHelp() []keybind.Keybind {
	return []keybind.Keybind{k.SelectUp.Keybind, k.SelectDown.Keybind, k.Select.Keybind, k.Cancel.Keybind}
}

func (k PickerKeybinds) FullHelp() [][]keybind.Keybind {
	return [][]keybind.Keybind{
		{k.SelectUp.Keybind, k.SelectDown.Keybind, k.SelectTop.Keybind, k.SelectBottom.Keybind},
		{k.Select.Keybind, k.Cancel.Keybind},
	}
}

var _ help.KeyMap = inboxKeyMap{}

// inboxKeyMap is the help of the inbox: a picker's keys with marking read after them.
type inboxKeyMap struct {
	picker PickerKeybinds
	inbox  InboxKeybinds
}

// InboxKeyMap returns the help that applies while the inbox is shown.
func (k Keybinds) InboxKeyMap() help.KeyMap { return inboxKeyMap{k.Picker, k.Inbox} }

func (k inboxKeyMap) ShortHelp() []keybind.Keybind {
	return append(k.picker.ShortHelp(), k.inbox.MarkRead.Keybind)
}

func (k inboxKeyMap) FullHelp() [][]keybind.Keybind {
	return append(k.picker.FullHelp(), []keybind.Keybind{k.inbox.MarkRead.Keybind, k.inbox.MarkAllRead.Keybind})
}
