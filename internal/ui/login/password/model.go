package password

import (
	"github.com/ayn2op/discordo/internal/ui/login/form"
	"github.com/ayn2op/tview"
)

type Model struct {
	form.Model
}

var _ tview.Model[Model] = Model{}

func NewModel() Model {
	return Model{Model: form.New("Login", form.Field{Label: "Login"}, form.Field{Label: "Password", Mask: "*"})}
}

func (Model) Label() string {
	return "Password"
}

func (m Model) Update(msg tview.Msg) (Model, tview.Cmd) {
	if _, ok := msg.(form.SubmitMsg); ok {
		login, password := m.Value(0), m.Value(1)
		if login == "" || password == "" {
			return m, nil
		}
		return m, submitLogin(login, password)
	}
	var cmd tview.Cmd
	m.Model, cmd = m.Model.Update(msg)
	return m, cmd
}
