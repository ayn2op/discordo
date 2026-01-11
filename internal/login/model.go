package login

import (
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/ayn2op/discordo/internal/config"
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
	if l.body == nil || headerArea.Max.Y >= area.Max.Y {
		return
	}

	bodyArea := area
	bodyArea.Min.Y = headerArea.Max.Y
	l.body.Draw(scr, bodyArea)
}

type Model struct {
	tabs   []*tab
	active int
	keys   keys
	cfg    *config.Config
}

func NewModel(cfg *config.Config) Model {
	return Model{
		tabs: []*tab{
			{"Token", newTokenModel()},
			{"Password", newPasswordModel()},
			{"QR", newQRModel()},
		},
		keys: defaultKeys(),
		cfg:  cfg,
	}
}

var _ tea.Model = Model{}

func (m Model) Init() tea.Cmd {
	if m.active >= len(m.tabs) {
		return nil
	}

	return m.tabs[m.active].model.Init()
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		k := msg.Key()
		switch {
		case key.Matches(k, m.keys.Previous):
			m.active = max(m.active-1, 0)
			return m, m.tabs[m.active].model.Init()
		case key.Matches(k, m.keys.Next):
			m.active = min(m.active+1, len(m.tabs)-1)
			return m, m.tabs[m.active].model.Init()
		}

	}

	var cmd tea.Cmd
	m.tabs[m.active].model, cmd = m.tabs[m.active].model.Update(msg)
	return m, tea.Batch(cmd)
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
