package ui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

var loginFormBackgroundColor = tcell.GetColor("#1C1E26")
var loginFormButtonBackgroundColor = tcell.GetColor("#5865F2")

func NewLoginForm(via string, onLoginFormLoginButtonSelected func(), onLoginFormQuitButtonSelected func()) *tview.Form {
	loginForm := tview.NewForm().
		AddButton("Login", onLoginFormLoginButtonSelected).
		AddButton("Quit", onLoginFormQuitButtonSelected)
	loginForm.
		SetButtonBackgroundColor(loginFormButtonBackgroundColor).
		SetBackgroundColor(loginFormBackgroundColor).
		SetBorder(true).
		SetBorderPadding(15, 15, 15, 15)

	if via == "token" {
		loginForm.AddPasswordField("Token", "", 0, 0, nil)
	} else if via == "emailpassword" {
		loginForm.
			AddInputField("Email", "", 0, nil, nil).
			AddPasswordField("Password", "", 0, 0, nil)
	}

	return loginForm
}
