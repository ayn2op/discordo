package main

import (
	"github.com/ayntgl/discordgo"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/zalando/go-keyring"
)

const Service = "discordo"

var (
	app               *tview.Application
	loginForm         *tview.Form
	guildsTreeView    *tview.TreeView
	messagesTextView  *tview.TextView
	messageInputField *tview.InputField
	mainFlex          *tview.Flex

	conf *config

	session         *discordgo.Session
	selectedChannel *discordgo.Channel
	selectedMessage *discordgo.Message
)

func main() {
	conf = loadConfig()
	tview.Styles = conf.Theme

	app = tview.NewApplication()
	app.
		EnableMouse(conf.Mouse).
		SetInputCapture(onAppInputCapture)

	guildsTreeView = newGuildsTreeView()
	messagesTextView = newMessagesTextView()
	messageInputField = newMessageInputField()

	rightFlex := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(messagesTextView, 0, 1, false).
		AddItem(messageInputField, 3, 1, false)
	mainFlex = tview.NewFlex().
		AddItem(guildsTreeView, 0, 1, false).
		AddItem(rightFlex, 0, 4, false)

	token := conf.Token
	if t, _ := keyring.Get(Service, "token"); t != "" {
		token = t
	}

	if token != "" {
		app.
			SetRoot(mainFlex, true).
			SetFocus(guildsTreeView)

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

func onAppInputCapture(e *tcell.EventKey) *tcell.EventKey {
	switch e.Name() {
	case conf.Keybindings.GuildsTreeViewFocus:
		app.SetFocus(guildsTreeView)
	case conf.Keybindings.MessagesTextViewFocus:
		app.SetFocus(messagesTextView)
	case conf.Keybindings.MessageInputFieldFocus:
		app.SetFocus(messageInputField)
	}

	return e
}

func onLoginFormLoginButtonSelected() {
	email := loginForm.GetFormItem(0).(*tview.InputField).GetText()
	password := loginForm.GetFormItem(1).(*tview.InputField).GetText()
	if email == "" || password == "" {
		return
	}

	session = newSession()
	// Login using the email and password
	lr, err := login(session, email, password)
	if err != nil {
		panic(err)
	}

	if lr.Token != "" && !lr.MFA {
		app.
			SetRoot(mainFlex, true).
			SetFocus(guildsTreeView)

		session.Token = lr.Token
		session.Identify.Token = lr.Token
		if err = session.Open(); err != nil {
			panic(err)
		}

		go keyring.Set(Service, "token", lr.Token)
	} else if lr.MFA {
		// The account has MFA enabled, reattempt login with code and ticket.
		loginForm = newLoginForm(func() {
			code := loginForm.GetFormItem(0).(*tview.InputField).GetText()
			if code == "" {
				return
			}

			lr, err = totp(session, code, lr.Ticket)
			if err != nil {
				panic(err)
			}

			app.
				SetRoot(mainFlex, true).
				SetFocus(guildsTreeView)

			session.Token = lr.Token
			session.Identify.Token = lr.Token
			if err = session.Open(); err != nil {
				panic(err)
			}

			go keyring.Set(Service, "token", lr.Token)
		}, true)

		app.SetRoot(loginForm, true)
	}
}
