package login

import (
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
)

type TokenModel struct {
	input textinput.Model
}

func newTokenModel() *TokenModel {
	input := textinput.New()
	input.Placeholder = "Token"
	input.EchoMode = textinput.EchoPassword
	input.Focus()
	return &TokenModel{
		input: input,
	}
}

var _ tea.Model = TokenModel{}

func (m TokenModel) Init() tea.Cmd {
	return m.input.Focus()
}

func (m TokenModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m TokenModel) View() tea.View {
	return tea.NewView(m.input.View())
}
