package chat

import (
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
		{cfg.ToggleMute.Keybind, cfg.ToggleDeafen.Keybind},
		{cfg.Logout.Keybind},
	}
}

func (m Model) focusHelp() []keybind.Keybind {
	kbs := m.cfg.Keybinds
	focused := m.focused
	focusKbs := make([]keybind.Keybind, 0, 4)

	if focused != paneGuildsTree {
		focusKbs = append(focusKbs, kbs.FocusGuildsTree.Keybind)
	}
	if focused != paneMessagesList {
		focusKbs = append(focusKbs, kbs.FocusMessagesList.Keybind)
	}
	if !m.composer.Disabled() && focused != paneComposer {
		focusKbs = append(focusKbs, kbs.FocusComposer.Keybind)
	}
	if m.membersTreeShown() && focused != paneMembersTree {
		focusKbs = append(focusKbs, kbs.FocusMembersTree.Keybind)
	}

	return focusKbs
}
