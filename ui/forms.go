package ui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func newBaseLoginForm() *tview.Form {
	f := tview.NewForm()
	f.
		SetButtonsAlign(tview.AlignCenter).
		SetButtonBackgroundColor(tcell.GetColor("#5865F2")).
		SetBorder(true).
		SetBorderPadding(0, 0, 1, 0)

	return f
}

// NewLoginForm creates and returns a new login form.
func NewLoginForm(onLoginFormLoginButtonSelected func()) *tview.Form {
	f := newBaseLoginForm()
	f.
		AddInputField("Email", "", 0, nil, nil).
		AddPasswordField("Password", "", 0, 0, nil).
		AddButton("Login", onLoginFormLoginButtonSelected)

	return f
}

// NewMfaLoginForm creates and returns a new MFA login form.
func NewMfaLoginForm(onMfaLoginFormLoginButtonSelected func()) *tview.Form {
	f := newBaseLoginForm().
		AddPasswordField("Code", "", 0, 0, nil).
		AddButton("Login", onMfaLoginFormLoginButtonSelected)

	return f
}
