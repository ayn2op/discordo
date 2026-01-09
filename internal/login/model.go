package login

import tea "charm.land/bubbletea/v2"

type Model struct{}

func NewModel() Model {
	return Model{}
}

var _ tea.Model = Model{}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(tea.Msg) (tea.Model, tea.Cmd) {
	return m, nil
}

func (m Model) View() tea.View {
	return tea.NewView("login")
}
