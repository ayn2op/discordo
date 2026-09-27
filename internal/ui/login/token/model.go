package token

import (
	"github.com/ayn2op/discordo/internal/ui/login/form"
	"github.com/ayn2op/tview"
)

type Model struct {
	form.Model
}

var _ tview.Model[Model] = Model{}

func NewModel() Model {
	return Model{Model: form.New("Login", form.Field{Label: "Token", Mask: "*"})}
}

func (Model) Label() string {
	return "Token"
}

func (m Model) Update(msg tview.Msg) (Model, tview.Cmd) {
	if _, ok := msg.(form.SubmitMsg); ok {
		token := m.Value(0)
		if token == "" {
			return m, nil
		}
		return m, submitToken(token)
	}
	var cmd tview.Cmd
	m.Model, cmd = m.Model.Update(msg)
	return m, cmd
}
