package tabs

import (
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type Tab interface {
	tea.Model
	Name() string
}

type Model struct {
	Keybinds Keybinds
	Styles   Styles

	tabs   []Tab
	active int

	width, height int
}

func NewModel(tabs []Tab) Model {
	return Model{
		Keybinds: DefaultKeybinds(),
		Styles:   DefaultStyles(),
		tabs:     tabs,
	}
}

func (m Model) Init() tea.Cmd {
	var cmd tea.Cmd
	if m.active < len(m.tabs) {
		cmd = m.tabs[m.active].Init()
	}
	return tea.Batch(tea.RequestWindowSize, cmd)
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if len(m.tabs) == 0 {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height

	case tea.KeyMsg:
		k := msg.Key()
		switch {
		case key.Matches(k, m.Keybinds.Previous):
			m.active = max(m.active-1, 0)
			return m, m.tabs[m.active].Init()
		case key.Matches(k, m.Keybinds.Next):
			m.active = min(m.active+1, len(m.tabs)-1)
			return m, m.tabs[m.active].Init()
		}
	}

	var (
		tabModel tea.Model
		cmd      tea.Cmd
	)
	tabModel, cmd = m.tabs[m.active].Update(msg)
	if tab, ok := tabModel.(Tab); ok {
		m.tabs[m.active] = tab
	}
	return m, cmd
}

func (m Model) View() tea.View {
	if len(m.tabs) == 0 {
		return tea.NewView("")
	}

	var content string
	tabLabels := make([]string, len(m.tabs))
	for index, tab := range m.tabs {
		var style lipgloss.Style
		if index == m.active {
			style = m.Styles.ActiveTab
			content = tab.View().Content
		} else {
			style = m.Styles.InactiveTab
		}

		tabLabels[index] = style.Render(tab.Name())
	}

	tabRow := lipgloss.JoinHorizontal(lipgloss.Top, tabLabels...)
	column := lipgloss.JoinVertical(
		lipgloss.Top,
		lipgloss.NewStyle().Width(m.width).Align(lipgloss.Center).Render(tabRow),
		content,
	)
	return tea.NewView(column)
}
