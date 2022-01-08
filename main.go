package main

import (
	"os"

	"github.com/ayntgl/discordgo"
	"github.com/ayntgl/discordo/config"
	"github.com/ayntgl/discordo/ui"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/zalando/go-keyring"
)

const service = "discordo"

var (
	app               *ui.App
	loginForm         *tview.Form
	channelsTreeView  *tview.TreeView
	messagesTextView  *tview.TextView
	messageInputField *tview.InputField
	mainFlex          *tview.Flex

	selectedChannel *discordgo.Channel
	selectedMessage int = -1
)

func main() {
	app = ui.NewApp()
	app.
		EnableMouse(config.General.Mouse).
		SetInputCapture(onAppInputCapture)

	app.ChannelsTreeView.
		SetTopLevel(1).
		SetRoot(tview.NewTreeNode("")).
		SetSelectedFunc(onChannelsTreeSelected).
		SetTitleAlign(tview.AlignLeft).
		SetBorder(true).
		SetBorderPadding(0, 0, 1, 0)

	app.MessagesTextView.
		SetRegions(true).
		SetDynamicColors(true).
		SetWordWrap(true).
		SetChangedFunc(func() { app.Draw() }).
		SetTitleAlign(tview.AlignLeft).
		SetInputCapture(onMessagesViewInputCapture).
		SetBorder(true).
		SetBorderPadding(0, 0, 1, 0)

	app.MessageInputField.
		SetPlaceholder("Message...").
		SetPlaceholderTextColor(tcell.ColorWhite).
		SetFieldBackgroundColor(tview.Styles.PrimitiveBackgroundColor).
		SetInputCapture(onMessageInputFieldInputCapture).
		SetTitleAlign(tview.AlignLeft).
		SetBorder(true).
		SetBorderPadding(0, 0, 1, 0)

	rightFlex := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(messagesTextView, 0, 1, false).
		AddItem(messageInputField, 3, 1, false)
	mainFlex = tview.NewFlex().
		AddItem(channelsTreeView, 0, 1, false).
		AddItem(rightFlex, 0, 4, false)

	token := os.Getenv("DISCORDO_TOKEN")
	if token == "" {
		token, _ = keyring.Get(service, "token")
	}

	if token != "" {
		app.
			SetRoot(mainFlex, true).
			SetFocus(channelsTreeView)

		app.Session.AddHandlerOnce(onSessionReady)
		app.Session.AddHandler(onSessionMessageCreate)
		err := app.Connect(token)
		if err != nil {
			panic(err)
		}
	} else {
		loginForm = ui.NewLoginForm(onLoginFormLoginButtonSelected, false)
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

	// Login using the email and password
	lr, err := login(email, password)
	if err != nil {
		panic(err)
	}

	if lr.Token != "" && !lr.MFA {
		app.
			SetRoot(mainFlex, true).
			SetFocus(channelsTreeView)

		app.Session.AddHandlerOnce(onSessionReady)
		app.Session.AddHandler(onSessionMessageCreate)
		err = app.Connect(lr.Token)
		if err != nil {
			panic(err)
		}

		go keyring.Set(service, "token", lr.Token)
	} else if lr.MFA {
		// The account has MFA enabled, reattempt login with code and ticket.
		loginForm = ui.NewLoginForm(func() {
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
				SetFocus(channelsTreeView)

			app.Session.AddHandlerOnce(onSessionReady)
			app.Session.AddHandler(onSessionMessageCreate)
			err = app.Connect(lr.Token)
			if err != nil {
				panic(err)
			}

			go keyring.Set(service, "token", lr.Token)
		}, true)

		app.SetRoot(loginForm, true)
	}
}
