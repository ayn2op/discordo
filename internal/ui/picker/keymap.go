package picker

import (
	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/tview"
	"github.com/ayn2op/tview/keybind"
	"github.com/ayn2op/tview/picker"
)

func pickerKeybind(keybinds config.PickerKeybinds) func(tview.KeyMsg) picker.Action {
	return func(key tview.KeyMsg) picker.Action {
		switch {
		case keybind.Matches(key, keybinds.Select.Keybind):
			return picker.ActionSelect
		case keybind.Matches(key, keybinds.Cancel.Keybind):
			return picker.ActionCancel
		}
		return picker.ActionNone
	}
}
