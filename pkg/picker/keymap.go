package picker

import "github.com/ayn2op/tview/keybind"

type KeyMap struct {
	Cancel keybind.Keybind

	Up     keybind.Keybind
	Down   keybind.Keybind
	Top    keybind.Keybind
	Bottom keybind.Keybind
	Select keybind.Keybind
}
