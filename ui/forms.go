package ui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func NewLoginForm(onLoginFormLoginButtonSelected func()) (f *tview.Form) {
	f = tview.NewForm()
	f.
		AddInputField("Email", "", 0, nil, nil).
		AddPasswordField("Password", "", 0, 0, nil).
		AddButton("Login", onLoginFormLoginButtonSelected).
		SetButtonsAlign(tview.AlignCenter).
		SetButtonBackgroundColor(tcell.GetColor("#5865F2")).
		SetBorder(true).
		SetBorderPadding(0, 0, 1, 1)

	return f
}
