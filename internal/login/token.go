package login

import (
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"github.com/ayn2op/discordo/form"
)

type TokenModel struct {
	form *form.Model
}

func newTokenModel() TokenModel {
	input := textinput.New()
	input.Placeholder = "Token"
	input.EchoMode = textinput.EchoPassword

	return TokenModel{
		form: form.NewModel([]textinput.Model{input}),
	}
}

var _ tea.Model = TokenModel{}

func (m TokenModel) Init() tea.Cmd {
	return m.form.Init()
}

func (m TokenModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.form, cmd = m.form.Update(msg)
	return m, cmd
}

func (m TokenModel) View() tea.View {
	return m.form.View()
}
