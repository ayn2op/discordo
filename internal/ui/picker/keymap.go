package picker

import (
	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/tview"
	"github.com/ayn2op/tview/keybind"
	"github.com/ayn2op/tview/picker"
)

func pickerKeybind(kbs config.PickerKeybinds) func(tview.KeyMsg) picker.Action {
	return func(key tview.KeyMsg) picker.Action {
		switch {
		case keybind.Matches(key, kbs.Select.Keybind):
			return picker.ActionSelect
		case keybind.Matches(key, kbs.Cancel.Keybind):
			return picker.ActionCancel
		}
		return picker.ActionNone
	}
}
