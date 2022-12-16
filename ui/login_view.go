package ui

import (
	"context"
	"log"

	"github.com/ayn2op/discordo/config"
	"github.com/diamondburned/arikawa/v3/api"
	"github.com/rivo/tview"
	"github.com/zalando/go-keyring"
)

type LoginView struct {
	*tview.Form
	app *Application
}

func newLoginView(app *Application) *LoginView {
	v := &LoginView{
		Form: tview.NewForm(),
		app:  app,
	}

	v.AddInputField("Email", "", 0, nil, nil)
	v.AddPasswordField("Password", "", 0, 0, nil)
	v.AddPasswordField("Code (optional)", "", 0, 0, nil)
	v.AddButton("Login", v.onLoginButtonSelected)

	v.SetTitle("Login")
	v.SetTitleAlign(tview.AlignLeft)
	v.SetBorder(true)
	v.SetBorderPadding(1, 1, 1, 1)

	return v
}

func (v *LoginView) onLoginButtonSelected() {
	email := v.GetFormItem(0).(*tview.InputField).GetText()
	password := v.GetFormItem(1).(*tview.InputField).GetText()
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

	// If the token is not dispatched in the response and the "mfa" field is set as true, login using MFA instead.
	if l.Token == "" && l.MFA {
		code := v.GetFormItem(2).(*tview.InputField).GetText()
		if code == "" {
			return
		}

		// Retry logging in with a 2FA token
		l, err = client.TOTP(code, l.Ticket)
		if err != nil {
			log.Fatal(err)
		}
	}

	err = v.app.Connect(l.Token)
	if err != nil {
		log.Fatal(err)
	}

	v.app.SetRoot(v.app.view, true)
	v.app.SetFocus(v.app.view.GuildsView)

	go keyring.Set(config.Name, "token", l.Token)
}
