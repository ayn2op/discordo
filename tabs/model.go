package tabs

import (
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type Tab struct {
	Name string
	tea.Model
}

func NewTab(name string, model tea.Model) Tab {
	return Tab{name, model}
}

type Model struct {
	KeyMap KeyMap
	Styles Styles

	tabs   []Tab
	active int
}

func NewModel(tabs []Tab) Model {
	return Model{
		KeyMap: DefaultKeyMap(),
		Styles: DefaultStyles(),
		tabs:   tabs,
	}
}

func (m Model) Init() tea.Cmd {
	if m.active < len(m.tabs) {
		return m.tabs[m.active].Init()
	}
	return nil
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		k := msg.Key()
		switch {
		case key.Matches(k, m.KeyMap.Previous):
			m.active = max(m.active-1, 0)
			return m, m.tabs[m.active].Init()
		case key.Matches(k, m.KeyMap.Next):
			m.active = min(m.active+1, len(m.tabs)-1)
			return m, m.tabs[m.active].Init()
		}
	}

	var cmd tea.Cmd
	m.tabs[m.active].Model, cmd = m.tabs[m.active].Update(msg)
	return m, cmd
}

func (m Model) View() tea.View {
	var contentView tea.View
	names := make([]string, len(m.tabs))
	for i, t := range m.tabs {
		var style lipgloss.Style
		if i == m.active {
			style = m.Styles.ActiveTab
			contentView = t.View()
		} else {
			style = m.Styles.InactiveTab
		}

		names[i] = style.Render(t.Name)
	}

	row := lipgloss.JoinHorizontal(lipgloss.Top, names...)
	return tea.NewView(stackedLayer{header: row, body: contentView})
}

type stackedLayer struct {
	header string
	body   tea.View
}

func (l stackedLayer) Draw(scr tea.Screen, area tea.Rectangle) {
	width := area.Max.X - area.Min.X
	header := l.header
	if width > 0 && header != "" {
		header = lipgloss.Place(width, lipgloss.Height(header), lipgloss.Center, lipgloss.Top, header)
	}

	headerView := tea.NewView(header)
	headerHeight := lipgloss.Height(header)
	headerArea := area
	headerArea.Max.Y = min(area.Min.Y+headerHeight, area.Max.Y)
	headerView.Content.Draw(scr, headerArea)
	if headerArea.Max.Y >= area.Max.Y {
		return
	}

	bodyArea := area
	bodyArea.Min.Y = headerArea.Max.Y
	l.body.Content.Draw(scr, bodyArea)
}
