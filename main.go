package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"

	"github.com/ayntgl/discordo/ui"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	lua "github.com/yuin/gopher-lua"
	"github.com/zalando/go-keyring"
)

const (
	name = "discordo"
)

var (
	token      string
	configPath string
)

func init() {
	flag.StringVar(&token, "token", "", "The client authentication token.")
	// If the token is provided via a command-line flag, store it in the default keyring.
	if token != "" {
		go keyring.Set(name, "token", token)
	}

	if token == "" {
		token, _ = keyring.Get(name, "token")
	}

	configDirPath, err := os.UserConfigDir()
	if err != nil {
		log.Fatal(err)
	}

	flag.StringVar(&configPath, "config", filepath.Join(configDirPath, name), "The path to the configuration directory.")
}

func main() {
	flag.Parse()

	c := ui.NewCore(configPath)
	if token != "" {
		err := c.Run(token)
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
				err = c.Run(lr.Token)
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

					err = c.Run(lr.Token)
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

	err := c.Config.Load()
	if err != nil {
		return
	}

	themeTable, ok := c.Config.State.GetGlobal("theme").(*lua.LTable)
	if !ok {
		themeTable = c.Config.State.NewTable()
	}

	background := themeTable.RawGetString("background")
	border := themeTable.RawGetString("border")
	title := themeTable.RawGetString("title")

	tview.Styles.PrimitiveBackgroundColor = tcell.GetColor(lua.LVAsString(background))
	tview.Styles.BorderColor = tcell.GetColor(lua.LVAsString(border))
	tview.Styles.TitleColor = tcell.GetColor(lua.LVAsString(title))

	err = c.Application.Run()
	if err != nil {
		log.Fatal(err)
	}
}
