package token

import (
	"github.com/ayn2op/discordo/internal/ui/login/form"
	"github.com/ayn2op/tview"
)

type Model struct {
	*form.Model
}

func NewModel() *Model {
	return &Model{Model: form.New("Login", form.Field{Label: "Token", Mask: "*"})}
}

func (m *Model) Label() string {
	return "Token"
}

func (m *Model) Update(msg tview.Msg) tview.Cmd {
	if _, ok := msg.(form.SubmitMsg); ok {
		token := m.Value(0)
		if token == "" {
			return nil
		}
		return submitToken(token)
	}
	return m.Model.Update(msg)
}
