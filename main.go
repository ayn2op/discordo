package main

import (
	"github.com/alecthomas/kong"
	"github.com/ayntgl/discordo/config"
	"github.com/ayntgl/discordo/discord"
	"github.com/ayntgl/discordo/ui"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/zalando/go-keyring"
)

const name = "discordo"

func main() {
	cfg := config.New()
	kong.Parse(cfg,
		kong.Name(name),
		kong.UsageOnError(),
		kong.Vars{
			"config": config.DefaultPath(),
		},
	)

	if cfg.Token == "" {
		cfg.Token, _ = keyring.Get(name, "token")
	}

	app := ui.NewApp(cfg)
	if cfg.Token != "" {
		err := app.Connect(cfg.Token)
		if err != nil {
			panic(err)
		}

		app.DrawMainFlex()
		app.SetFocus(app.GuildsList)
	} else {
		loginForm := ui.NewLoginForm(false)
		loginForm.AddButton("Login", func() {
			email := loginForm.GetFormItem(0).(*tview.InputField).GetText()
			password := loginForm.GetFormItem(1).(*tview.InputField).GetText()
			if email == "" || password == "" {
				return
			}

			// Login using the email and password
			lr, err := discord.Login(app.Session, email, password)
			if err != nil {
				panic(err)
			}

			if lr.Token != "" && !lr.MFA {
				err = app.Connect(lr.Token)
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
					mfaCode := loginForm.GetFormItem(0).(*tview.InputField).GetText()
					if mfaCode == "" {
						return
					}

					lr, err = discord.TOTP(app.Session, mfaCode, lr.Ticket)
					if err != nil {
						panic(err)
					}

					err = app.Connect(lr.Token)
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
