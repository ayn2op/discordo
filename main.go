package main

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/zalando/go-keyring"
)

const service = "discordo"

func main() {
	conf = loadConfig()

	tview.Styles.PrimitiveBackgroundColor = tcell.GetColor(conf.Theme.PrimitiveBackgroundColor)
	tview.Styles.ContrastBackgroundColor = tcell.GetColor(conf.Theme.ContrastBackgroundColor)
	tview.Styles.MoreContrastBackgroundColor = tcell.GetColor(conf.Theme.MoreContrastBackgroundColor)
	tview.Styles.BorderColor = tcell.GetColor(conf.Theme.BorderColor)
	tview.Styles.TitleColor = tcell.GetColor(conf.Theme.TitleColor)
	tview.Styles.GraphicsColor = tcell.GetColor(conf.Theme.GraphicsColor)
	tview.Styles.PrimaryTextColor = tcell.GetColor(conf.Theme.PrimaryTextColor)
	tview.Styles.SecondaryTextColor = tcell.GetColor(conf.Theme.SecondaryTextColor)
	tview.Styles.TertiaryTextColor = tcell.GetColor(conf.Theme.TertiaryTextColor)
	tview.Styles.InverseTextColor = tcell.GetColor(conf.Theme.InverseTextColor)
	tview.Styles.ContrastSecondaryTextColor = tcell.GetColor(conf.Theme.ContrastSecondaryTextColor)

	app = newApplication()
	channelsTree = newChannelsTree()
	messagesTextView = newMessagesTextView()
	messageInputField = newMessageInputField()

	rightFlex := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(messagesTextView, 0, 1, false).
		AddItem(messageInputField, 3, 1, false)
	mainFlex = tview.NewFlex().
		AddItem(channelsTree, 0, 1, false).
		AddItem(rightFlex, 0, 4, false)

	token := conf.Token
	if t, _ := keyring.Get(service, "token"); t != "" {
		token = t
	}

	if token != "" {
		app.
			SetRoot(mainFlex, true).
			SetFocus(channelsTree)

		session = newSession()
		session.Token = token
		session.Identify.Token = token
		if err := session.Open(); err != nil {
			panic(err)
		}
	} else {
		loginForm = newLoginForm(onLoginFormLoginButtonSelected, false)
		app.SetRoot(loginForm, true)
	}

	if err := app.Run(); err != nil {
		panic(err)
	}
}

func onLoginFormLoginButtonSelected() {
	email := loginForm.GetFormItem(0).(*tview.InputField).GetText()
	password := loginForm.GetFormItem(1).(*tview.InputField).GetText()
	if email == "" || password == "" {
		return
	}

	session = newSession()
	// Login using the email and password
	lr, err := login(email, password)
	if err != nil {
		panic(err)
	}

	if lr.Token != "" && !lr.MFA {
		app.
			SetRoot(mainFlex, true).
			SetFocus(channelsTree)

		session.Token = lr.Token
		session.Identify.Token = lr.Token
		if err = session.Open(); err != nil {
			panic(err)
		}

		go keyring.Set(service, "token", lr.Token)
	} else if lr.MFA {
		// The account has MFA enabled, reattempt login with code and ticket.
		loginForm = newLoginForm(func() {
			code := loginForm.GetFormItem(0).(*tview.InputField).GetText()
			if code == "" {
				return
			}

			lr, err = totp(code, lr.Ticket)
			if err != nil {
				panic(err)
			}

			app.
				SetRoot(mainFlex, true).
				SetFocus(channelsTree)

			session.Token = lr.Token
			session.Identify.Token = lr.Token
			if err = session.Open(); err != nil {
				panic(err)
			}

			go keyring.Set(service, "token", lr.Token)
		}, true)

		app.SetRoot(loginForm, true)
	}
}
