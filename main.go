package main

import (
	"os"

	"github.com/ayntgl/discordo/ui"
	"github.com/ayntgl/discordo/util"
	"github.com/rivo/tview"
	"github.com/zalando/go-keyring"
)

func main() {
	app := ui.NewApp()

	token := os.Getenv("DISCORDO_TOKEN")
	if token == "" {
		token, _ = keyring.Get("discordo", "token")
	}

	if token != "" {
		err := app.Connect(token)
		if err != nil {
			panic(err)
		}

		app.
			SetRoot(ui.NewMainFlex(app), true).
			SetFocus(app.GuildsList)
	} else {
		loginForm := ui.NewLoginForm(false)
		loginForm.AddButton("Login", func() {
			email := loginForm.GetFormItem(0).(*tview.InputField).GetText()
			password := loginForm.GetFormItem(1).(*tview.InputField).GetText()
			if email == "" || password == "" {
				return
			}

			// Login using the email and password
			lr, err := util.Login(app.Session, email, password)
			if err != nil {
				panic(err)
			}

			if lr.Token != "" && !lr.MFA {
				err = app.Connect(lr.Token)
				if err != nil {
					panic(err)
				}

				app.
					SetRoot(ui.NewMainFlex(app), true).
					SetFocus(app.GuildsList)

				go keyring.Set("discordo", "token", lr.Token)
			} else {
				// The account has MFA enabled, reattempt login with MFA code and ticket.
				mfaLoginForm := ui.NewLoginForm(true)
				mfaLoginForm.AddButton("Login", func() {
					mfaCode := loginForm.GetFormItem(0).(*tview.InputField).GetText()
					if mfaCode == "" {
						return
					}

					lr, err = util.TOTP(app.Session, mfaCode, lr.Ticket)
					if err != nil {
						panic(err)
					}

					err = app.Connect(lr.Token)
					if err != nil {
						panic(err)
					}

					app.
						SetRoot(ui.NewMainFlex(app), true).
						SetFocus(app.GuildsList)

					go keyring.Set("discordo", "token", lr.Token)
				})
			}
		})

		app.SetRoot(loginForm, true)
	}

	if err := app.EnableMouse(app.Config.General.Mouse).Run(); err != nil {
		panic(err)
	}
}
