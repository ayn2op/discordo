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
	login, password string
	// loginCursor and passwordCursor are the byte offsets of the cursor in login and password.
	loginCursor, passwordCursor int

	focus focus
}

var _ tview.Model[Model] = Model{}

func (Model) Label() string {
	return "Password"
}

func (Model) Init() tview.Cmd { return nil }

func (m Model) Update(msg tview.Msg) (Model, tview.Cmd) {
	switch msg := msg.(type) {
	case loginMsg:
		m.login, m.loginCursor = msg.text, msg.cursor
	case passwordMsg:
		m.password, m.passwordCursor = msg.text, msg.cursor
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
		if m.login == "" || m.password == "" {
			return m, nil
		}
		return m, submitLogin(m.login, m.password)
	}
	return m, nil
}

func (m Model) View() tview.Widget {
	labelWidth := layout.Fixed(len("Password") + 1)
	login := textinput.New(m.login, m.loginCursor).
		Focused(m.focus == focusLogin).
		OnInput(func(text string, cursor int) tview.Msg { return loginMsg{text, cursor} }).
		OnSubmit(focusMsg(focusPassword))
	password := textinput.New(m.password, m.passwordCursor).
		Mask("*").
		Focused(m.focus == focusPassword).
		OnInput(func(text string, cursor int) tview.Msg { return passwordMsg{text, cursor} }).
		OnSubmit(focusMsg(focusButton))
	submit := button.New().
		Label("Login").
		Width(layout.Fixed(len("Login") + 4)).
		Height(layout.Fixed(1)).
		Focused(m.focus == focusButton).
		OnClick(submitMsg{})
	return column.New(
		row.New(column.New(text.New("Login")).Width(labelWidth), login).Height(layout.Shrink),
		row.New(column.New(text.New("Password")).Width(labelWidth), password).Height(layout.Shrink),
		submit,
	).Spacing(1).Align(tview.AlignmentCenter)
}
