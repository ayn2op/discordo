package login

import (
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"github.com/ayn2op/discordo/form"
	"github.com/ayn2op/discordo/internal/keyring"
)

type TokenMsg struct {
	Value string
	Err   error
}

func tokenMsgCmd(value string, err error) tea.Cmd {
	return func() tea.Msg {
		return TokenMsg{Value: value, Err: err}
	}
}

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

func (m TokenModel) Name() string {
	return "Token"
}

var _ tea.Model = TokenModel{}

func (m TokenModel) Init() tea.Cmd {
	return m.form.Init()
}

func (m TokenModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.form.Submitted {
		token := m.form.Get(0).Value()
		if err := keyring.SetToken(token); err != nil {
			return m, tokenMsgCmd("", err)
		}
		return m, tokenMsgCmd(token, nil)
	}

	var cmd tea.Cmd
	m.form, cmd = m.form.Update(msg)
	return m, cmd
}

func (m TokenModel) View() tea.View {
	return m.form.View()
}
