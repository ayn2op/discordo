package login

import (
	"github.com/ayn2op/tview/help"
	"github.com/ayn2op/tview/keybind"
)

var _ help.KeyMap = Model{}

func (m Model) ShortHelp() []keybind.Keybind {
	short := m.navHelp()
	if keyMap := m.activeKeyMap(); keyMap != nil {
		short = append(short, keyMap.ShortHelp()...)
	}
	return short
}

func (m Model) FullHelp() [][]keybind.Keybind {
	var full [][]keybind.Keybind
	if nav := m.navHelp(); len(nav) > 0 {
		full = append(full, nav)
	}
	if keyMap := m.activeKeyMap(); keyMap != nil {
		full = append(full, keyMap.FullHelp()...)
	}
	return full
}

// navHelp returns the tab keybinds that switch to another tab from the active one.
func (m Model) navHelp() []keybind.Keybind {
	var nav []keybind.Keybind
	if m.active > 0 {
		nav = append(nav, m.keybinds.Previous)
	}
	if m.active < tabCount-1 {
		nav = append(nav, m.keybinds.Next)
	}
	return nav
}

// activeKeyMap returns the keybinds of the active tab, or nil if it has none.
func (m Model) activeKeyMap() help.KeyMap {
	var active any
	switch m.active {
	case passwordTab:
		active = m.password
	case qrTab:
		active = m.qr
	case tokenTab:
		active = m.token
	}
	keyMap, _ := active.(help.KeyMap)
	return keyMap
}
