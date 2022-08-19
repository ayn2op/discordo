package main

import (
	"log"

	"github.com/alecthomas/kong"
	"github.com/ayntgl/discordo/config"
	"github.com/ayntgl/discordo/ui"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/zalando/go-keyring"
)

const (
	name  = "discordo"
	usage = "A lightweight, secure, and feature-rich Discord terminal client"
)

var cli struct {
	Token  string `help:"The authentication token."`
	Config string `help:"The path of the configuration file." type:"path"`
}

func main() {
	kong.Parse(&cli, kong.Name(name), kong.Description(usage), kong.UsageOnError())

	// If the authentication token is provided via a flag, store it in the default keyring.
	if cli.Token != "" {
		go keyring.Set(name, "token", cli.Token)
	}

	// Defaults
	if cli.Config == "" {
		cli.Config = config.DefaultPath()
	}

	if cli.Token == "" {
		cli.Token, _ = keyring.Get(name, "token")
	}

	c := config.New()
	err := c.Load(cli.Config)
	if err != nil {
		log.Fatal(err)
	}

	app := ui.NewApp(cli.Token, c)
	if cli.Token != "" {
		err := app.Connect()
		if err != nil {
			log.Fatal(err)
		}

		app.DrawMainFlex()

		app.SetRoot(app.MainFlex, true)
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
				app.SetRoot(app.MainFlex, true)
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
					app.SetRoot(app.MainFlex, true)
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

	err = app.Run()
	if err != nil {
		log.Fatal(err)
	}
}
