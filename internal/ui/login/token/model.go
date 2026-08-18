package token

import (
	"github.com/ayn2op/tview"
	"github.com/ayn2op/tview/tabs"
)

type Model struct {
	*tview.Form
}

func NewModel() *Model {
	form := tview.NewForm().
		AddPasswordField("Token", "", 0, 0).
		AddButton("Login")
	return &Model{Form: form}
}

var _ tabs.Tab = (*Model)(nil)

func (m *Model) Label() string {
	return "Token"
}

func (m *Model) Update(msg tview.Msg) tview.Cmd {
	if _, ok := msg.(tview.FormSubmitMsg); ok {
		token := m.GetFormItem(0).(*tview.InputField).Text()
		if token == "" {
			return nil
		}
		return tokenCmd(token)
	}
	return m.Form.Update(msg)
}
