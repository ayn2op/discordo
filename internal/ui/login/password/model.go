package password

import (
	"github.com/ayn2op/tview"
	"github.com/ayn2op/tview/tabs"
)

type Model struct {
	*tview.Form
}

func NewModel() *Model {
	form := tview.NewForm().
		AddInputField("Login", "", 0).
		AddPasswordField("Password", "", 0, 0).
		AddButton("Login")
	return &Model{Form: form}
}

var _ tabs.Tab = (*Model)(nil)

func (m *Model) Label() string {
	return "Password"
}

func (m *Model) Update(msg tview.Msg) tview.Cmd {
	if _, ok := msg.(tview.FormSubmitMsg); ok {
		login := m.GetFormItem(0).(*tview.InputField).Text()
		password := m.GetFormItem(1).(*tview.InputField).Text()
		if login == "" || password == "" {
			return nil
		}
		return loginCmd(login, password)
	}
	return m.Form.Update(msg)
}
