package root

import (
	"github.com/ayn2op/tview/help"
	"github.com/ayn2op/tview/keybind"
)

var _ help.KeyMap = (*Model)(nil)

func (m *Model) ShortHelp() []keybind.Keybind {
	global := []keybind.Keybind{
		m.cfg.Keybinds.ToggleHelp.Keybind,
		m.cfg.Keybinds.Suspend.Keybind,
		m.cfg.Keybinds.Quit.Keybind,
	}
	if active := m.activeKeyMap(); active != nil {
		short := active.ShortHelp()
		return append(short, global...)
	}
	return global
}

func (m *Model) FullHelp() [][]keybind.Keybind {
	global := []keybind.Keybind{
		m.cfg.Keybinds.ToggleHelp.Keybind,
		m.cfg.Keybinds.Suspend.Keybind,
		m.cfg.Keybinds.Quit.Keybind,
	}
	if active := m.activeKeyMap(); active != nil {
		full := active.FullHelp()
		return append(full, global)
	}
	return [][]keybind.Keybind{global}
}

func (m *Model) activeKeyMap() help.KeyMap {
	if keyMap, ok := m.inner.(help.KeyMap); ok {
		return keyMap
	}
	return nil
}
