package chat

import (
	tea "charm.land/bubbletea/v2"
	"github.com/ayn2op/discordo/internal/config"
)

type Model struct {
	cfg   *config.Config
	token string
}

func NewModel(cfg *config.Config, token string) Model {
	return Model{cfg: cfg, token: token}
}

var _ tea.Model = Model{}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(tea.Msg) (tea.Model, tea.Cmd) {
	return m, nil
}

func (m Model) View() tea.View {
	return tea.NewView(m.token)
}
