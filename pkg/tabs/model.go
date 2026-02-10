package tabs

import (
	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type Tab interface {
	tea.Model
	Label() string
}

type Model struct {
	Keybinds Keybinds
	Styles   Styles

	tabs   []Tab
	active int

	windowSize tea.WindowSizeMsg
}

func NewModel(tabs []Tab) Model {
	return Model{
		Keybinds: DefaultKeybinds(),
		Styles:   DefaultStyles(),

		tabs: tabs,
	}
}

func (m *Model) Previous() {
	m.active = max(m.active-1, 0)
}

func (m *Model) Next() {
	m.active = min(m.active+1, len(m.tabs)-1)
}

func (m Model) Init() tea.Cmd {
	if len(m.tabs) == 0 {
		return nil
	}
	return m.tabs[m.active].Init()
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if len(m.tabs) == 0 {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.windowSize = msg
	case tea.KeyMsg:
		k := msg.Key()
		switch {
		case key.Matches(k, m.Keybinds.Previous):
			m.Previous()
			return m, m.activateTab()
		case key.Matches(k, m.Keybinds.Next):
			m.Next()
			return m, m.activateTab()
		}
	}

	updated, cmd := m.tabs[m.active].Update(msg)
	if tabModel, ok := updated.(Tab); ok {
		m.tabs[m.active] = tabModel
	}
	return m, cmd
}

func (m Model) activateTab() tea.Cmd {
	size := m.windowSize
	return tea.Batch(
		m.tabs[m.active].Init(),
		func() tea.Msg {
			return size
		},
	)
}

func (m Model) View() tea.View {
	if len(m.tabs) == 0 {
		return tea.NewView("")
	}

	tabHeader := m.renderTabHeader()
	tabView := m.tabs[m.active].View()
	content := lipgloss.PlaceHorizontal(m.windowSize.Width, lipgloss.Center, tabView.Content)
	bodyH := max(m.windowSize.Height-lipgloss.Height(tabHeader), 0)
	content = lipgloss.PlaceVertical(bodyH, lipgloss.Center, content)
	tabView.Content = lipgloss.JoinVertical(lipgloss.Left, tabHeader, content)
	return tabView
}

func (m Model) renderTabHeader() string {
	tabs := make([]string, len(m.tabs))
	for i, tab := range m.tabs {
		if i == m.active {
			tabs[i] = m.Styles.ActiveTabStyle.Render(tab.Label())
		} else {
			tabs[i] = m.Styles.TabStyle.Render(tab.Label())
		}
	}
	return m.Styles.TabLineStyle.Width(m.windowSize.Width).Render(lipgloss.JoinHorizontal(lipgloss.Top, tabs...))
}

var _ help.KeyMap = Model{}

func (m Model) ShortHelp() []key.Binding {
	short := []key.Binding{m.Keybinds.Previous, m.Keybinds.Next}
	if len(m.tabs) == 0 {
		return short
	}
	if activeKeyMap, ok := m.tabs[m.active].(help.KeyMap); ok {
		short = append(short, activeKeyMap.ShortHelp()...)
	}
	return short
}

func (m Model) FullHelp() [][]key.Binding {
	full := [][]key.Binding{{m.Keybinds.Previous, m.Keybinds.Next}}
	if len(m.tabs) == 0 {
		return full
	}
	if activeKeyMap, ok := m.tabs[m.active].(help.KeyMap); ok {
		full = append(full, activeKeyMap.FullHelp()...)
	}
	return full
}
