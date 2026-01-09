package login

import (
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

var (
	inactiveTabStyle = lipgloss.NewStyle().Padding(0, 1)
	activeTabStyle   = inactiveTabStyle.Background(lipgloss.Blue)
)

type tab struct {
	name  string
	model tea.Model
}

type stackedLayer struct {
	header string
	body   tea.Layer
}

func (l stackedLayer) Draw(scr tea.Screen, area tea.Rectangle) {
	headerView := tea.NewView(l.header)
	headerHeight := lipgloss.Height(l.header)
	headerArea := area
	headerArea.Max.Y = min(area.Min.Y+headerHeight, area.Max.Y)
	headerView.Content.Draw(scr, headerArea)
	if l.body == nil || headerArea.Max.Y >= area.Max.Y {
		return
	}

	bodyArea := area
	bodyArea.Min.Y = headerArea.Max.Y
	l.body.Draw(scr, bodyArea)
}

type Model struct {
	tabs   []tab
	active int
	keys   keys
}

func NewModel() Model {
	return Model{
		tabs: []tab{
			{"Token", newTokenModel()},
			{"Password", newPasswordModel()},
			{"QR", newQRModel()},
		},
		keys: defaultKeys(),
	}
}

var _ tea.Model = Model{}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		k := msg.Key()
		switch {
		case key.Matches(k, m.keys.Previous):
			m.active = max(m.active-1, 0)
		case key.Matches(k, m.keys.Next):
			m.active = min(m.active+1, len(m.tabs)-1)
		}
	}

	var cmd tea.Cmd
	tab := m.tabs[m.active]
	tab.model, cmd = tab.model.Update(msg)
	m.tabs[m.active] = tab
	return m, cmd
}

func (m Model) View() tea.View {
	var content tea.Layer
	names := make([]string, len(m.tabs))
	for i, t := range m.tabs {
		var style lipgloss.Style
		if i == m.active {
			style = activeTabStyle
			content = t.model.View().Content
		} else {
			style = inactiveTabStyle
		}

		names[i] = style.Render(t.name)
	}

	row := lipgloss.JoinHorizontal(lipgloss.Top, names...)
	return tea.NewView(stackedLayer{header: row, body: content})
}
