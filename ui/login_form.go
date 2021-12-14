package ui

import "github.com/rivo/tview"

func NewLoginForm(onLoginFormLoginButtonSelected func(), mfa bool) *tview.Form {
	f := tview.NewForm()
	f.
		AddButton("Login", onLoginFormLoginButtonSelected).
		SetButtonsAlign(tview.AlignCenter).
		SetBorder(true).
		SetBorderPadding(0, 0, 1, 0)

	if mfa {
		f.AddPasswordField("Code", "", 0, 0, nil)
	} else {
		f.
			AddInputField("Email", "", 0, nil, nil).
			AddPasswordField("Password", "", 0, 0, nil)
	}

	return f
}
