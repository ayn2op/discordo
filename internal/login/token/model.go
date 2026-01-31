package token

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

func tokenCmd(token string, err error) tea.Cmd {
	return func() tea.Msg {
		return TokenMsg{token, err}
	}
}

type Model struct {
	form *form.Model
}

func NewModel() Model {
	input := textinput.New()
	input.Placeholder = "Token"
	input.EchoMode = textinput.EchoPassword
	return Model{
		form: form.NewModel([]textinput.Model{input}),
	}
}

func (m Model) Name() string {
	return "Token"
}

var _ tea.Model = Model{}

func (m Model) Init() tea.Cmd {
	return m.form.Init()
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
			return m, tokenCmd("", err)
		}
		return m, tokenCmd(token, nil)
	}

	var cmd tea.Cmd
	m.form, cmd = m.form.Update(msg)
	return m, cmd
}

func (m Model) View() tea.View {
	return m.form.View()
}
