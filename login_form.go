package main

import (
	"context"
	"log"

	"github.com/diamondburned/arikawa/api"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type LoginForm struct {
	*tview.Form
}

func newLoginForm() *LoginForm {
	lf := &LoginForm{
		Form: tview.NewForm(),
	}

	lf.AddInputField("Email", "", 0, nil, nil)
	lf.AddPasswordField("Password", "", 0, 0, nil)
	lf.AddPasswordField("Code (optional)", "", 0, 0, nil)
	lf.AddButton("Login", lf.onLoginButtonSelected)

	lf.SetTitle("Login")
	lf.SetTitleColor(tcell.GetColor(cfg.Theme.TitleColor))

	p := cfg.Theme.BorderPadding
	lf.SetBorder(cfg.Theme.Border)
	lf.SetBorderColor(tcell.GetColor(cfg.Theme.BorderColor))
	lf.SetBorderPadding(p[0], p[1], p[2], p[3])

	return lf
}

func (lf *LoginForm) onLoginButtonSelected() {
	email := lf.GetFormItem(0).(*tview.InputField).GetText()
	password := lf.GetFormItem(1).(*tview.InputField).GetText()
	if email == "" || password == "" {
		return
	}

	// Make a scratch HTTP client without a token
	client := api.NewClient("").WithContext(context.Background())
	// Try to login without TOTP
	l, err := client.Login(email, password)
	if err != nil {
		log.Fatal(err)
	}

	// Retry logging in with a 2FA token
	if l.Token == "" && l.MFA {
		code := lf.GetFormItem(2).(*tview.InputField).GetText()
		if code == "" {
			return
		}

		l, err = client.TOTP(code, l.Ticket)
		if err != nil {
			log.Fatal(err)
		}
	}

	// We got the token, return with a new Session.
	discordState = newState(l.Token)
	err = discordState.Open(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	mainFlex = newMainFlex()
	app.SetRoot(mainFlex, true)
}
