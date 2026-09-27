package login

import (
	"github.com/ayn2op/tview/help"
	"github.com/ayn2op/tview/keybind"
)

var _ help.KeyMap = (*Model)(nil)

func (m *Model) ShortHelp() []keybind.Keybind {
	short := m.navHelp()
	if keyMap, ok := m.tabs[m.active].(help.KeyMap); ok {
		short = append(short, keyMap.ShortHelp()...)
	}
	return short
}

func (m *Model) FullHelp() [][]keybind.Keybind {
	var full [][]keybind.Keybind
	if nav := m.navHelp(); len(nav) > 0 {
		full = append(full, nav)
	}
	if keyMap, ok := m.tabs[m.active].(help.KeyMap); ok {
		full = append(full, keyMap.FullHelp()...)
	}
	return full
}

// navHelp returns the tab keybinds that switch to another tab from the active one.
func (m *Model) navHelp() []keybind.Keybind {
	var nav []keybind.Keybind
	if m.active > 0 {
		nav = append(nav, m.keybinds.Previous)
	}
	if m.active < len(m.tabs)-1 {
		nav = append(nav, m.keybinds.Next)
	}
	return nav
}
