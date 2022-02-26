package ui

import "github.com/rivo/tview"

type LoginForm struct {
	*tview.Form
}

func NewLoginForm(mfa bool) *LoginForm {
	lf := &LoginForm{
		Form: tview.NewForm(),
	}

	if mfa {
		lf.AddPasswordField("MFA Code (optional)", "", 0, 0, nil)
	} else {
		lf.
			AddInputField("Email", "", 0, nil, nil).
			AddPasswordField("Password", "", 0, 0, nil)
	}

	lf.SetButtonsAlign(tview.AlignCenter)
	lf.SetTitle("Login")
	lf.SetTitleAlign(tview.AlignLeft)
	lf.SetBorder(true)
	lf.SetBorderPadding(0, 0, 1, 1)
	return lf
}
