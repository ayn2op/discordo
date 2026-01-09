package login

import tea "charm.land/bubbletea/v2"

type PasswordModel struct{}

func newPasswordModel() PasswordModel {
	return PasswordModel{}
}

var _ tea.Model = PasswordModel{}

func (m PasswordModel) Init() tea.Cmd {
	return nil
}

func (m PasswordModel) Update(tea.Msg) (tea.Model, tea.Cmd) {
	return m, nil
}

func (m PasswordModel) View() tea.View {
	return tea.NewView("password")
}
