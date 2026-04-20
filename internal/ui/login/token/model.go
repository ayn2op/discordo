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
		AddPasswordField("Token", "", 0, 0, nil).
		AddButton("Login")
	return &Model{Form: form}
}

var _ tabs.Tab = (*Model)(nil)

func (m *Model) Label() string {
	return "Token"
}

func (m *Model) Update(msg tview.Msg) tview.Cmd {
	switch msg.(type) {
	case tview.FormSubmitMsg:
		token := m.GetFormItem(0).(*tview.InputField).GetText()
		if token == "" {
			return nil
		}
		return tokenCommand(token)
	}
	return m.Form.Update(msg)
}
