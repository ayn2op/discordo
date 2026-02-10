package token

import (
	"strings"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"github.com/ayn2op/discordo/pkg/form"
	"github.com/ayn2op/discordo/pkg/tabs"
)

type Model struct {
	form form.Model
}

func NewModel() Model {
	input := textinput.New()
	input.Placeholder = "Token"
	input.EchoMode = textinput.EchoPassword
	return Model{
		form: form.NewModel([]textinput.Model{input}),
	}
}

var _ tabs.Tab = Model{}

func (m Model) Label() string {
	return "Token"
}

var _ tea.Model = Model{}

func (m Model) Init() tea.Cmd {
	return m.form.Init()
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case form.SubmitMsg:
		value := strings.TrimSpace(msg.Values[0])
		if value == "" {
			return m, nil
		}
		return m, tokenCmd(value)
	}

	var cmd tea.Cmd
	m.form, cmd = m.form.Update(msg)
	return m, cmd
}

func (m Model) View() tea.View {
	return m.form.View()
}

var _ help.KeyMap = Model{}

func (m Model) ShortHelp() []key.Binding {
	return m.form.ShortHelp()
}

func (m Model) FullHelp() [][]key.Binding {
	return m.form.FullHelp()
}
