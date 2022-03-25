package main

import (
	"flag"

	"github.com/ayntgl/astatine"
	"github.com/ayntgl/discordo/config"
	"github.com/ayntgl/discordo/ui"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/zalando/go-keyring"
)

const name = "discordo"

func main() {
	var token string
	var cfg string
	flag.StringVar(&token, "token", "", "The authentication token.")
	flag.StringVar(&cfg, "config", "", "The path of the configuration file.")
	flag.Parse()

	if token == "" {
		token, _ = keyring.Get(name, "token")
	}

	c := config.New()
	if cfg != "" {
		c.Path = cfg
	}
	c.Load()

	app := ui.NewApp(c)
	if token != "" {
		app.Session = astatine.New(token)
		err := app.Connect()
		if err != nil {
			panic(err)
		}

		app.DrawMainFlex()
		app.SetFocus(app.GuildsList)
	} else {
		app.Session = astatine.New(token)

		loginForm := ui.NewLoginForm(false)
		loginForm.AddButton("Login", func() {
			email := loginForm.GetFormItem(0).(*tview.InputField).GetText()
			password := loginForm.GetFormItem(1).(*tview.InputField).GetText()
			if email == "" || password == "" {
				return
			}

			// Login using the email and password
			lr, err := app.Session.Login(email, password)
			if err != nil {
				panic(err)
			}

			if lr.Token != "" && !lr.Mfa {
				app.Session.Identify.Token = lr.Token
				err = app.Connect()
				if err != nil {
					panic(err)
				}

				app.DrawMainFlex()
				app.SetFocus(app.GuildsList)

				go keyring.Set(name, "token", lr.Token)
			} else {
				// The account has MFA enabled, reattempt login with MFA code and ticket.
				mfaLoginForm := ui.NewLoginForm(true)
				mfaLoginForm.AddButton("Login", func() {
					code := loginForm.GetFormItem(0).(*tview.InputField).GetText()
					if code == "" {
						return
					}

					lr, err = app.Session.Totp(code, lr.Ticket)
					if err != nil {
						panic(err)
					}

					app.Session.Identify.Token = lr.Token
					err = app.Connect()
					if err != nil {
						panic(err)
					}

					app.DrawMainFlex()
					app.SetFocus(app.GuildsList)

					go keyring.Set(name, "token", lr.Token)
				})
			}
		})

		app.SetRoot(loginForm, true)
	}

	tview.Borders.TopLeftFocus = tview.Borders.TopLeft
	tview.Borders.TopRightFocus = tview.Borders.TopRight
	tview.Borders.BottomLeftFocus = tview.Borders.BottomLeft
	tview.Borders.BottomRightFocus = tview.Borders.BottomRight
	tview.Borders.HorizontalFocus = tview.Borders.Horizontal
	tview.Borders.VerticalFocus = tview.Borders.Vertical
	tview.Borders.TopLeft = 0
	tview.Borders.TopRight = 0
	tview.Borders.BottomLeft = 0
	tview.Borders.BottomRight = 0
	tview.Borders.Horizontal = 0
	tview.Borders.Vertical = 0

	tview.Styles.PrimitiveBackgroundColor = tcell.GetColor(app.Config.Theme.Background)
	tview.Styles.BorderColor = tcell.GetColor(app.Config.Theme.Border)
	tview.Styles.TitleColor = tcell.GetColor(app.Config.Theme.Title)

	err := app.Run()
	if err != nil {
		panic(err)
	}
}
