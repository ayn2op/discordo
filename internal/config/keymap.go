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
