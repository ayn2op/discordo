package login

import (
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"github.com/ayn2op/discordo/internal/keyring"
	"github.com/ayn2op/discordo/pkg/form"
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
	switch msg := msg.(type) {
	case form.SubmitMsg:
		if len(msg.Values) < 1 {
			return m, nil
		}
		token := strings.TrimSpace(msg.Values[0])
		if token == "" {
			return m, nil
		}
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
