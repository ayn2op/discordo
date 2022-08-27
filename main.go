package main

import (
	"log"
	"os"
	"path/filepath"

	"github.com/alecthomas/kong"
	"github.com/ayntgl/discordo/ui"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	lua "github.com/yuin/gopher-lua"
	"github.com/zalando/go-keyring"
)

const (
	name  = "discordo"
	usage = "A lightweight, secure, and feature-rich Discord terminal client"
)

var cli struct {
	Token  string `help:"The authentication token."`
	Config string `help:"The path to the configuration file." type:"path"`
}

func main() {
	kong.Parse(&cli, kong.Name(name), kong.Description(usage), kong.UsageOnError())

	// If the authentication token is provided via a flag, store it in the default keyring.
	if cli.Token != "" {
		go keyring.Set(name, "token", cli.Token)
	}

	// Defaults
	if cli.Config == "" {
		path, err := os.UserConfigDir()
		if err != nil {
			log.Fatal(err)
		}

		cli.Config = filepath.Join(path, "discordo.lua")
	}

	if cli.Token == "" {
		cli.Token, _ = keyring.Get(name, "token")
	}

	c := ui.NewCore(cli.Token, cli.Config)
	if cli.Token != "" {
		err := c.Run()
		if err != nil {
			log.Fatal(err)
		}

		c.DrawMainFlex()
		c.Application.SetRoot(c.MainFlex, true)
		c.Application.SetFocus(c.GuildsTree)
	} else {
		loginForm := ui.NewLoginForm(false)
		loginForm.AddButton("Login", func() {
			email := loginForm.GetFormItem(0).(*tview.InputField).GetText()
			password := loginForm.GetFormItem(1).(*tview.InputField).GetText()
			if email == "" || password == "" {
				return
			}

			// Login using the email and password only
			lr, err := c.State.Login(email, password)
			if err != nil {
				log.Fatal(err)
			}

			if lr.Token != "" && !lr.MFA {
				c.State.Token = lr.Token
				err = c.Run()
				if err != nil {
					log.Fatal(err)
				}

				c.DrawMainFlex()
				c.Application.SetRoot(c.MainFlex, true)
				c.Application.SetFocus(c.GuildsTree)

				go keyring.Set(name, "token", lr.Token)
			} else {
				// The account has MFA enabled, reattempt login with MFA code and ticket.
				mfaLoginForm := ui.NewLoginForm(true)
				mfaLoginForm.AddButton("Login", func() {
					code := mfaLoginForm.GetFormItem(0).(*tview.InputField).GetText()
					if code == "" {
						return
					}

					lr, err = c.State.TOTP(code, lr.Ticket)
					if err != nil {
						log.Fatal(err)
					}

					c.State.Token = lr.Token
					err = c.Run()
					if err != nil {
						log.Fatal(err)
					}

					c.DrawMainFlex()
					c.Application.SetRoot(c.MainFlex, true)
					c.Application.SetFocus(c.GuildsTree)

					go keyring.Set(name, "token", lr.Token)
				})

				c.Application.SetRoot(mfaLoginForm, true)
			}
		})

		c.Application.SetRoot(loginForm, true)
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

	themeTable, ok := c.LState.GetGlobal("theme").(*lua.LTable)
	if !ok {
		return
	}

	background := themeTable.RawGetString("background")
	border := themeTable.RawGetString("border")
	title := themeTable.RawGetString("title")

	tview.Styles.PrimitiveBackgroundColor = tcell.GetColor(lua.LVAsString(background))
	tview.Styles.BorderColor = tcell.GetColor(lua.LVAsString(border))
	tview.Styles.TitleColor = tcell.GetColor(lua.LVAsString(title))

	err := c.Application.Run()
	if err != nil {
		log.Fatal(err)
	}
}
