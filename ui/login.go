package ui

import "github.com/rivo/tview"

const (
	EmailInputFieldLabel    = "Email"
	PasswordInputFieldLabel = "Password"
)

type LoginForm struct {
	*tview.Form
}

func NewLoginForm() *LoginForm {
	lf := &LoginForm{
		Form: tview.NewForm(),
	}

	lf.AddInputField(EmailInputFieldLabel, "", 0, nil, nil)
	lf.AddPasswordField(PasswordInputFieldLabel, "", 0, 0, nil)

	lf.SetTitle("Login")
	lf.SetTitleAlign(tview.AlignLeft)
	lf.SetBorder(true)
	lf.SetBorderPadding(0, 0, 1, 1)

	return lf
}
