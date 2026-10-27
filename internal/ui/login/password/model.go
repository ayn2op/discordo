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

// focus is the input or button that receives keys in the credentials stage.
type focus int

const (
	focusLogin focus = iota
	focusPassword
	focusButton
	focusCount
)

// codeFocus is the input or button that receives keys in the code stage.
const (
	focusCode focus = iota
	focusCodeButton
	focusCodeCount
)

// stage is the step of the login flow currently shown.
type stage int

const (
	stageCredentials stage = iota
	stageCode
)

// mfaMethod is the way the code in the code stage is verified.
type mfaMethod int

const (
	mfaMethodTOTP mfaMethod = iota
	mfaMethodSMS
)

// Model is a login and a password input above a button. Submitting them and finding a second factor required switches to a single code input above a button. Tab and the arrow keys move the focus, Enter in an input moves to the next one, and Enter on the button or clicking it submits.
type Model struct {
	login, password string
	// loginCursor and passwordCursor are the byte offsets of the cursor in login and password.
	loginCursor, passwordCursor int

	code       string
	codeCursor int

	stage  stage
	method mfaMethod
	// ticket and loginInstanceID identify the pending login once Discord asks for a code.
	ticket, loginInstanceID string

	focus focus
	// buttonHovered reports whether the pointer is over the button.
	buttonHovered bool
}

func (Model) Label() string {
	return "Password"
}

var _ tview.Model[Model] = Model{}

func (Model) Init() tview.Cmd { return nil }

func (m Model) Update(msg tview.Msg) (Model, tview.Cmd) {
	switch msg := msg.(type) {
	case loginMsg:
		m.login, m.loginCursor = msg.text, msg.cursor
	case passwordMsg:
		m.password, m.passwordCursor = msg.text, msg.cursor
	case codeMsg:
		m.code, m.codeCursor = msg.text, msg.cursor
	case focusMsg:
		m.focus = focus(msg)
	case hoverMsg:
		m.buttonHovered = bool(msg)
	case tview.KeyMsg:
		count := focusCount
		if m.stage == stageCode {
			count = focusCodeCount
		}
		switch msg.Key() {
		case tcell.KeyTab, tcell.KeyDown:
			m.focus = (m.focus + 1) % count
		case tcell.KeyBacktab, tcell.KeyUp:
			m.focus = (m.focus + count - 1) % count
		}
	case submitMsg:
		if m.stage == stageCode {
			return m, m.submitCode()
		}
		return m, m.submitCredentials()
	}
	return m, nil
}

// submitCredentials starts a login once both the login and password are filled in.
func (m Model) submitCredentials() tview.Cmd {
	if m.login == "" || m.password == "" {
		return nil
	}
	return submitLogin(m.login, m.password)
}

// submitCode verifies the code against the pending login once it is filled in.
func (m Model) submitCode() tview.Cmd {
	if m.code == "" {
		return nil
	}
	return verifyMFACode(m.method, m.code, m.ticket, m.loginInstanceID)
}

func (m Model) View() tview.Widget {
	if m.stage == stageCode {
		return m.codeView()
	}

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
	buttonStyle := tcell.StyleDefault
	switch {
	case m.focus == focusButton:
		buttonStyle = buttonStyle.Reverse(true)
	case m.buttonHovered:
		buttonStyle = buttonStyle.Underline(true)
	}
	submit := button.New(text.New("Login")).
		Style(buttonStyle).
		Width(layout.Fixed(len("Login") + 4)).
		Focused(m.focus == focusButton).
		Hovered(m.buttonHovered).
		OnHover(func(hovered bool) tview.Msg { return hoverMsg(hovered) }).
		OnClick(submitMsg{})
	return column.New(
		row.New(text.New("Login").Width(labelWidth), login).Height(layout.Shrink),
		row.New(text.New("Password").Width(labelWidth), password).Height(layout.Shrink),
		submit,
	).Spacing(1).AlignX(tview.HorizontalAlignmentCenter)
}

// codeView shows a single input for the authenticator or SMS code Discord asked for.
func (m Model) codeView() tview.Widget {
	label := "Authenticator code"
	if m.method == mfaMethodSMS {
		label = "SMS code"
	}

	labelWidth := layout.Fixed(len(label) + 1)
	code := textinput.New(m.code, m.codeCursor).
		Focused(m.focus == focusCode).
		OnInput(func(text string, cursor int) tview.Msg { return codeMsg{text, cursor} }).
		OnSubmit(focusMsg(focusCodeButton))
	submit := button.New(text.New("Submit")).
		Width(layout.Fixed(len("Submit") + 4)).
		Focused(m.focus == focusCodeButton).
		OnClick(submitMsg{})
	return column.New(
		row.New(text.New(label).Width(labelWidth), code).Height(layout.Shrink),
		submit,
	).Spacing(1).AlignX(tview.HorizontalAlignmentCenter)
}
