package ui

import (
	"context"
	"log"

	"github.com/ayntgl/discordo/config"
	"github.com/diamondburned/arikawa/v3/api"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/zalando/go-keyring"
)

const (
	emailViewPageName = "email"
	tokenViewPageName = "token"
)

func NewLoginView(c *Core) *tview.Pages {
	v := tview.NewPages()

	v.AddPage(emailViewPageName, newEmailView(c), true, true)
	v.AddPage(tokenViewPageName, newTokenView(c), true, true)
	// Since the recommended method to login is using the email and password, it is displayed on the screen first.
	v.SwitchToPage(emailViewPageName)

	v.SetTitle("Login")
	v.SetTitleAlign(tview.AlignLeft)
	v.SetBorder(true)
	v.SetBorderPadding(0, 0, 1, 1)
	v.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyCtrlSpace {
			name, _ := v.GetFrontPage()

			switch name {
			case emailViewPageName:
				name = tokenViewPageName
			case tokenViewPageName:
				name = emailViewPageName
			}

			v.SwitchToPage(name)
			return nil
		}

		return event
	})

	return v
}

type EmailView struct {
	*tview.Form
	core *Core
}

func newEmailView(c *Core) *EmailView {
	v := &EmailView{
		Form: tview.NewForm(),
		core: c,
	}

	v.AddInputField("Email", "", 0, nil, nil)
	v.AddPasswordField("Password", "", 0, 0, nil)
	v.AddPasswordField("Code (optional)", "", 0, 0, nil)
	v.AddButton("Login", v.onLoginButtonSelected)

	return v
}

func (v *EmailView) onLoginButtonSelected() {
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

	err = v.core.Run(l.Token)
	if err != nil {
		log.Fatal(err)
	}

	v.core.Draw()
	go keyring.Set(config.Name, "token", l.Token)
}

type TokenView struct {
	*tview.Form
	core *Core
}

func newTokenView(c *Core) *TokenView {
	v := &TokenView{
		Form: tview.NewForm(),
		core: c,
	}

	v.AddPasswordField("Token", "", 0, 0, nil)
	v.AddButton("Login", v.onLoginButtonSelected)

	return v
}

func (v *TokenView) onLoginButtonSelected() {
	token := v.GetFormItem(0).(*tview.InputField).GetText()
	if token == "" {
		return
	}

	err := v.core.Run(token)
	if err != nil {
		log.Fatal(err)
	}

	v.core.Draw()
	go keyring.Set(config.Name, "token", token)
}
