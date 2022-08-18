package main

import (
	"log"
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
		err := c.Load(ctx.String("config"))
		if err != nil {
			return err
		}

		token := ctx.String("token")
		app := ui.NewApp(token, c)
		if token != "" {
			err := app.Connect()
			if err != nil {
				return err
			}

			app.DrawMainFlex()
			app.SetFocus(app.GuildsTree)
		} else {
			loginForm := ui.NewLoginForm(false)
			loginForm.AddButton("Login", func() {
				email := loginForm.GetFormItem(0).(*tview.InputField).GetText()
				password := loginForm.GetFormItem(1).(*tview.InputField).GetText()
				if email == "" || password == "" {
					return
				}

				// Login using the email and password only
				lr, err := app.State.Login(email, password)
				if err != nil {
					log.Fatal(err)
				}

				if lr.Token != "" && !lr.MFA {
					app.State.Token = lr.Token
					err = app.Connect()
					if err != nil {
						log.Fatal(err)
					}

					app.DrawMainFlex()
					app.SetFocus(app.GuildsTree)

					go keyring.Set(name, "token", lr.Token)
				} else {
					// The account has MFA enabled, reattempt login with MFA code and ticket.
					mfaLoginForm := ui.NewLoginForm(true)
					mfaLoginForm.AddButton("Login", func() {
						code := mfaLoginForm.GetFormItem(0).(*tview.InputField).GetText()
						if code == "" {
							return
						}

						lr, err = app.State.TOTP(code, lr.Ticket)
						if err != nil {
							log.Fatal(err)
						}

						app.State.Token = lr.Token
						err = app.Connect()
						if err != nil {
							log.Fatal(err)
						}

						app.DrawMainFlex()
						app.SetFocus(app.GuildsTree)

						go keyring.Set(name, "token", lr.Token)
					})

					app.SetRoot(mfaLoginForm, true)
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

		return app.Run()
	}

	err := cliApp.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
