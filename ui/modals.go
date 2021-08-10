package ui

import (
	"github.com/rivo/tview"
)

const (
	LoginViaTokenLoginModalButton         = "Login via token"
	LoginViaEmailPasswordLoginModalButton = "Login via email and password"
)

func NewLoginModal(onLoginModalDone func(buttonIndex int, buttonLabel string)) *tview.Modal {
	m := tview.NewModal()
	m.
		SetText("Choose a login method:").
		AddButtons([]string{LoginViaTokenLoginModalButton, LoginViaEmailPasswordLoginModalButton}).
		SetDoneFunc(onLoginModalDone).
		SetBorder(true).
		SetBorderPadding(0, 0, 1, 1)

	return m
}
