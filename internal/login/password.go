package login

import (
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"github.com/ayn2op/discordo/pkg/form"
)

type PasswordModel struct {
	form *form.Model
}

func newPasswordModel() PasswordModel {
	login := textinput.New()
	login.Placeholder = "Email or phone number"

	password := textinput.New()
	password.Placeholder = "Password"
	password.EchoMode = textinput.EchoPassword

	return PasswordModel{
		form: form.NewModel([]textinput.Model{login, password}),
	}
}

func (m PasswordModel) Name() string {
	return "Password"
}

var _ tea.Model = PasswordModel{}

func (m PasswordModel) Init() tea.Cmd {
	return m.form.Init()
}

func (m PasswordModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case form.SubmitMsg:
		// login input + password input
		if len(msg.Values) < 2 {
			return m, nil
		}
		login := strings.TrimSpace(msg.Values[0])
		password := strings.TrimSpace(msg.Values[1])
		if login == "" || password == "" {
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.form, cmd = m.form.Update(msg)
	return m, cmd
}

func (m PasswordModel) View() tea.View {
	return m.form.View()
}
