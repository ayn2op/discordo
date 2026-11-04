package chat

import (
	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/tview/help"
	"github.com/ayn2op/tview/keybind"
)

var _ help.KeyMap = Model{}

func (m Model) ShortHelp() []keybind.Keybind {
	short := make([]keybind.Keybind, 0, 16)
	if active := m.activeKeyMap(); active != nil {
		short = append(short, active.ShortHelp()...)
	}
	short = append(short, m.baseShortHelp()...)
	return short
}

func (m Model) FullHelp() [][]keybind.Keybind {
	full := make([][]keybind.Keybind, 0, 8)
	if active := m.activeKeyMap(); active != nil {
		full = append(full, active.FullHelp()...)
	}
	full = append(full, m.baseFullHelp()...)
	return full
}

func (m Model) activeKeyMap() help.KeyMap {
	if m.overlay != overlayNone {
		return m.cfg.Keybinds.Picker
	}

	switch m.focused {
	case paneGuildsTree:
		return m.guildsTree
	case paneMessagesList:
		return m.messagesList
	case paneComposer:
		return m.composer
	case paneMembersTree:
		return m.membersTree
	default:
		return nil
	}
}

func (m Model) baseShortHelp() []keybind.Keybind {
	cfg := m.cfg.Keybinds
	short := m.focusHelp()
	short = append(short, cfg.ToggleGuildsTree.Keybind, cfg.ToggleChannelsPicker.Keybind)
	return short
}

func (m Model) baseFullHelp() [][]keybind.Keybind {
	cfg := m.cfg.Keybinds
	return [][]keybind.Keybind{
		m.focusHelp(),
		{cfg.FocusPrevious.Keybind, cfg.FocusNext.Keybind},
		{cfg.ToggleGuildsTree.Keybind, cfg.ToggleMembersTree.Keybind, cfg.ToggleChannelsPicker.Keybind},
		{cfg.ToggleMute.Keybind, cfg.ToggleDeafen.Keybind, cfg.LeaveVoice.Keybind},
		{favoriteHelp(cfg.Favorites, "favorite"), favoriteHelp(cfg.SetFavorites, "set favorite")},
		{cfg.Logout.Keybind},
	}
}

// favoriteHelp describes every favorite slot as one entry, such as "ctrl+1-ctrl+0", since a line per
// slot would crowd the rest of the help out. The returned keybind is for help alone: the slots are
// matched against their own keybinds.
func favoriteHelp(keybinds []config.Keybind, desc string) keybind.Keybind {
	if len(keybinds) == 0 {
		return keybind.Keybind{}
	}

	first, last := keybinds[0].Help().Key, keybinds[len(keybinds)-1].Help().Key
	return keybind.New().WithHelp(first+"-"+last, desc)
}

func (m Model) focusHelp() []keybind.Keybind {
	keybinds := m.cfg.Keybinds
	focused := m.focused
	focusKeybinds := make([]keybind.Keybind, 0, 4)

	if focused != paneGuildsTree {
		focusKeybinds = append(focusKeybinds, keybinds.FocusGuildsTree.Keybind)
	}
	if focused != paneMessagesList {
		focusKeybinds = append(focusKeybinds, keybinds.FocusMessagesList.Keybind)
	}
	if !m.composer.Disabled() && focused != paneComposer {
		focusKeybinds = append(focusKeybinds, keybinds.FocusComposer.Keybind)
	}
	if m.membersTreeShown() && focused != paneMembersTree {
		focusKeybinds = append(focusKeybinds, keybinds.FocusMembersTree.Keybind)
	}

	return focusKeybinds
}
