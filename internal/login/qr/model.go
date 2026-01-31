package qr

import (
	tea "charm.land/bubbletea/v2"
	"github.com/ayn2op/discordo/pkg/tabs"
)

type Model struct{}

func NewModel() Model {
	return Model{}
}

var _ tabs.Tab = Model{}

func (m Model) Name() string {
	return "QR"
}

var _ tea.Model = Model{}

func (m Model) Init() tea.Cmd {
	panic("unimplemented")
}

func (m Model) Update(tea.Msg) (tea.Model, tea.Cmd) {
	panic("unimplemented")
}

func (m Model) View() tea.View {
	panic("unimplemented")
}
