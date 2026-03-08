package root

import (
	"github.com/ayn2op/tview/help"
	"github.com/ayn2op/tview/keybind"
)

var _ help.KeyMap = (*View)(nil)

func (v *View) ShortHelp() []keybind.Keybind {
	global := []keybind.Keybind{
		v.cfg.Keybinds.ToggleHelp.Keybind,
		v.cfg.Keybinds.Suspend.Keybind,
		v.cfg.Keybinds.Quit.Keybind,
	}
	if active := v.activeKeyMap(); active != nil {
		short := active.ShortHelp()
		return append(short, global...)
	}
	return global
}

func (v *View) FullHelp() [][]keybind.Keybind {
	global := []keybind.Keybind{
		v.cfg.Keybinds.ToggleHelp.Keybind,
		v.cfg.Keybinds.Suspend.Keybind,
		v.cfg.Keybinds.Quit.Keybind,
	}
	if active := v.activeKeyMap(); active != nil {
		full := active.FullHelp()
		return append(full, global)
	}
	return [][]keybind.Keybind{global}
}

func (v *View) activeKeyMap() help.KeyMap {
	if keyMap, ok := v.inner.(help.KeyMap); ok {
		return keyMap
	}
	return nil
}
