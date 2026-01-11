package login

import (
	tea "charm.land/bubbletea/v2"
	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/discordo/tabs"
)

type Model struct {
	tabs tabs.Model
	cfg  *config.Config
}

func NewModel(cfg *config.Config) Model {
	return Model{
		tabs: tabs.NewModel([]tabs.Tab{
			newTokenModel(),
			newPasswordModel(),
		}),
		cfg: cfg,
	}
}

var _ tea.Model = Model{}

func (m Model) Init() tea.Cmd {
	return m.tabs.Init()
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.tabs, cmd = m.tabs.Update(msg)
	return m, cmd
}

func (m Model) View() tea.View {
	view := m.tabs.View()
	view.WindowTitle = "Login"
	return view
}
