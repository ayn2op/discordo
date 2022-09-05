package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"

	"github.com/ayntgl/discordo/config"
	"github.com/ayntgl/discordo/ui"
	"github.com/rivo/tview"
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

	cfg := config.New(configPath)
	err := cfg.Load()
	if err != nil {
		log.Fatal(err)
	}

	c := ui.NewCore(cfg)
	if token != "" {
		err = c.Run(token)
		if err != nil {
			log.Fatal(err)
		}

		c.DrawMainFlex()
		c.Application.SetRoot(c.MainFlex, true)
		c.Application.SetFocus(c.GuildsTree)
	} else {
		loginForm := tview.NewForm()
		loginForm.AddPasswordField("Token", "", 0, 0, nil)
		loginForm.SetButtonsAlign(tview.AlignCenter)

		loginForm.SetTitle("Login")
		loginForm.SetTitleAlign(tview.AlignLeft)
		loginForm.SetBorder(true)
		loginForm.SetBorderPadding(0, 0, 1, 1)

		loginForm.AddButton("Login", func() {
			tkn := loginForm.GetFormItem(0).(*tview.InputField).GetText()
			if tkn == "" {
				return
			}

			err := c.Run(tkn)
			if err != nil {
				log.Fatal(err)
			}

			c.DrawMainFlex()
			c.Application.SetRoot(c.MainFlex, true)
			c.Application.SetFocus(c.GuildsTree)
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

	err = c.Application.Run()
	if err != nil {
		log.Fatal(err)
	}
}
