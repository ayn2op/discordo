package login

import (
	"log"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/diamondburned/arikawa/v3/api"
)

type TokenMsg string

type Model struct {
	form *huh.Form
}

func NewModel() Model {
	return Model{
		form: huh.NewForm(huh.NewGroup(
			huh.NewInput().
				Key("id").
				Placeholder("Email or Phone Number").
				Validate(huh.ValidateNotEmpty()),

			huh.NewInput().
				Key("password").
				Placeholder("Password").
				EchoMode(huh.EchoModePassword).
				Validate(huh.ValidateNotEmpty()),
		)),
	}
}

func (m Model) Init() tea.Cmd {
	return m.form.Init()
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m.form.State {
	case huh.StateCompleted:
		id := m.form.GetString("id")
		password := m.form.GetString("password")
		return m, login(id, password)
	}

	form, cmd := m.form.Update(msg)
	if f, ok := form.(*huh.Form); ok {
		m.form = f
	}

	return m, cmd
}

func (m Model) View() string {
	return m.form.View()
}

func login(id, password string) tea.Cmd {
	return func() tea.Msg {
		client := api.NewClient("")
		resp, err := client.Login(id, password)
		if err != nil {
			// TODO: send errMsg
			log.Fatal(err)
		}

		return TokenMsg(resp.Token)
	}
}
