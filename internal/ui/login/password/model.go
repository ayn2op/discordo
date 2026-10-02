package password

import (
	"github.com/ayn2op/tview"
	"github.com/ayn2op/tview/button"
	"github.com/ayn2op/tview/column"
	"github.com/ayn2op/tview/layout"
	"github.com/ayn2op/tview/row"
	"github.com/ayn2op/tview/text"
	"github.com/ayn2op/tview/textinput"
	"github.com/gdamore/tcell/v3"
)

// focus is the input or button that receives keys.
type focus int

const (
	focusLogin focus = iota
	focusPassword
	focusButton
	focusCount
)

// Model is a login and a password input above a button. Tab and the arrow keys move the focus, Enter in an input moves to the next one, and Enter on the button or clicking it logs in.
type Model struct {
	login, password textinput.EditState
	focus           focus
}

var _ tview.Model[Model] = Model{}

func (Model) Label() string {
	return "Password"
}

func (Model) Init() tview.Cmd { return nil }

func (m Model) Update(msg tview.Msg) (Model, tview.Cmd) {
	switch msg := msg.(type) {
	case loginMsg:
		m.login.Apply(textinput.Change(msg))
	case passwordMsg:
		m.password.Apply(textinput.Change(msg))
	case focusMsg:
		m.focus = focus(msg)
	case tview.KeyMsg:
		switch msg.Key() {
		case tcell.KeyTab, tcell.KeyDown:
			m.focus = (m.focus + 1) % focusCount
		case tcell.KeyBacktab, tcell.KeyUp:
			m.focus = (m.focus + focusCount - 1) % focusCount
		}
	case submitMsg:
		login, password := m.login.Value(), m.password.Value()
		if login == "" || password == "" {
			return m, nil
		}
		return m, submitLogin(login, password)
	}
	return m, nil
}

func (m Model) View() tview.Element {
	labelWidth := layout.Fixed(len("Password") + 1)
	login := textinput.New(&m.login).
		Focused(m.focus == focusLogin).
		OnChange(func(c textinput.Change) tview.Msg { return loginMsg(c) }).
		OnSubmit(focusMsg(focusPassword))
	password := textinput.New(&m.password).
		Mask("*").
		Focused(m.focus == focusPassword).
		OnChange(func(c textinput.Change) tview.Msg { return passwordMsg(c) }).
		OnSubmit(focusMsg(focusButton))
	submit := button.New().
		Label("Login").
		Width(layout.Fixed(len("Login") + 4)).
		Height(layout.Fixed(1)).
		Focused(m.focus == focusButton).
		OnClick(submitMsg{})
	return column.New(
		row.New(column.New(text.New("Login")).Width(labelWidth), login).Height(layout.Fixed(1)),
		row.New(column.New(text.New("Password")).Width(labelWidth), password).Height(layout.Fixed(1)),
		submit,
	).Spacing(1)
}
