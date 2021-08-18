package ui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func newBaseLoginForm() (f *tview.Form) {
	f = tview.NewForm()
	f.
		SetButtonsAlign(tview.AlignCenter).
		SetButtonBackgroundColor(tcell.GetColor("#5865F2")).
		SetBorder(true).
		SetBorderPadding(0, 0, 1, 0)

	return
}

// NewLoginForm creates and returns a new login form.
func NewLoginForm(onLoginFormLoginButtonSelected func()) (f *tview.Form) {
	f = newBaseLoginForm()
	f.
		AddInputField("Email", "", 0, nil, nil).
		AddPasswordField("Password", "", 0, 0, nil).
		AddButton("Login", onLoginFormLoginButtonSelected)

	return
}

// NewMfaLoginForm creates and returns a new MFA login form.
func NewMfaLoginForm(onMfaLoginFormLoginButtonSelected func()) (f *tview.Form) {
	f = newBaseLoginForm().
		AddPasswordField("Code", "", 0, 0, nil).
		AddButton("Login", onMfaLoginFormLoginButtonSelected)

	return
}
