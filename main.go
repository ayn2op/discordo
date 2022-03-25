package main

import (
	"os"

	"github.com/ayntgl/discordo/config"
	"github.com/ayntgl/discordo/ui"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/urfave/cli/v2"
	"github.com/zalando/go-keyring"
)

const (
	name  = "discordo"
	usage = "A lightweight, secure, and feature-rich Discord terminal client"
)

func main() {
	t, _ := keyring.Get(name, "token")

	cliApp := &cli.App{
		Name:                 name,
		Usage:                usage,
		EnableBashCompletion: true,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "token",
				Usage:       "The client authentication token.",
				Value:       t,
				DefaultText: "From keyring",
				Aliases:     []string{"t"},
			},
			&cli.StringFlag{
				Name:    "config",
				Usage:   "The path of the configuration file.",
				Value:   config.DefaultPath(),
				Aliases: []string{"c"},
			},
		},
	}

	cliApp.Action = func(ctx *cli.Context) error {
		c := config.New()
		c.Load(ctx.String("config"))

		token := ctx.String("token")
		app := ui.NewApp(token, c)
		if token != "" {
			err := app.Connect()
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

		return nil
	}

	err := cliApp.Run(os.Args)
	if err != nil {
		panic(err)
	}
}
