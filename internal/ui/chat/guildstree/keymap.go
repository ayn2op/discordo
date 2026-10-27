package guildstree

import (
	"slices"

	"github.com/ayn2op/arikawa/v3/discord"
	uitree "github.com/ayn2op/discordo/internal/ui/tree"
	"github.com/ayn2op/tview/help"
	"github.com/ayn2op/tview/keybind"
	"github.com/ayn2op/tview/tree"
)

var _ help.KeyMap = Model{}

func (m Model) selectCurrentKeybind() keybind.Keybind {
	selectCurrent := m.cfg.Keybinds.GuildsTree.SelectCurrent.Keybind
	selectHelp := selectCurrent.Help()
	return selectCurrent.WithHelp(selectHelp.Key, m.selectDesc(m.current, selectHelp.Desc))
}

// selectDesc returns what selecting node does, or desc if it does nothing more specific.
func (m Model) selectDesc(node *tree.Node, desc string) string {
	if node == nil {
		return desc
	}

	if m.expandable(node) {
		if node.Expanded() {
			return "collapse"
		}

		return "expand"
	}

	switch node.Reference().(type) {
	case discord.GuildID, dmNode:
		return "expand"
	}

	return desc
}

// toggleVoiceKeybinds returns the keybind that joins or leaves the highlighted voice channel, if it is one.
func (m Model) toggleVoiceKeybinds() []keybind.Keybind {
	channel := m.voiceChannel(m.current)
	if channel == nil {
		return nil
	}

	toggleVoice := m.cfg.Keybinds.GuildsTree.ToggleVoice.Keybind
	if m.voiceSession.Status().ChannelID == channel.ID {
		toggleVoice = toggleVoice.WithHelp(toggleVoice.Help().Key, "leave voice")
	}

	return []keybind.Keybind{toggleVoice}
}

func (m Model) ShortHelp() []keybind.Keybind {
	cfg := m.cfg.Keybinds.GuildsTree
	shortHelp := append(uitree.SelectHelp(m.root, m.current, cfg.TreeKeybinds), m.selectCurrentKeybind())
	shortHelp = append(shortHelp, m.toggleVoiceKeybinds()...)
	if m.canCollapseParent(m.current) {
		shortHelp = append(shortHelp, cfg.CollapseParentNode.Keybind)
	}
	return shortHelp
}

func (m Model) FullHelp() [][]keybind.Keybind {
	cfg := m.cfg.Keybinds.GuildsTree
	selectGroup := []keybind.Keybind{m.selectCurrentKeybind(), cfg.MoveToParentNode.Keybind}
	selectGroup = append(selectGroup, m.collapseKeybinds()...)

	return [][]keybind.Keybind{
		{cfg.SelectUp.Keybind, cfg.SelectDown.Keybind, cfg.SelectTop.Keybind, cfg.SelectBottom.Keybind},
		{cfg.ScrollUp.Keybind, cfg.ScrollDown.Keybind, cfg.ScrollTop.Keybind, cfg.ScrollBottom.Keybind, cfg.ScrollLeft.Keybind, cfg.ScrollRight.Keybind},
		selectGroup,
		{cfg.SelectPreviousUnread.Keybind, cfg.SelectNextUnread.Keybind},
		append([]keybind.Keybind{cfg.YankID.Keybind}, m.toggleVoiceKeybinds()...),
	}
}

func (m Model) collapseKeybinds() []keybind.Keybind {
	cfg := m.cfg.Keybinds.GuildsTree

	var keybinds []keybind.Keybind
	if m.canCollapseParent(m.current) {
		keybinds = append(keybinds, cfg.CollapseParentNode.Keybind)
	}
	if m.canCollapseAll() {
		keybinds = append(keybinds, cfg.CollapseAll.Keybind)
	}
	return keybinds
}

func (m Model) canCollapseParent(node *tree.Node) bool {
	return node != nil && len(m.root.PathTo(node)) >= 3
}

func (m Model) canCollapseAll() bool {
	return slices.ContainsFunc(m.root.Children(), (*tree.Node).Expanded)
}
