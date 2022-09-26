package main

import (
	"fmt"
	"log"
	"runtime"

	"github.com/ayntgl/discordo/config"
	"github.com/ayntgl/discordo/ui"
	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/rivo/tview"
	"github.com/zalando/go-keyring"
)

func init() {
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

	api.UserAgent = fmt.Sprintf("%s/%s %s/%s", config.Name, "0.1", "arikawa", "v3")
	gateway.DefaultIdentity = gateway.IdentifyProperties{
		OS:      runtime.GOOS,
		Browser: config.Name,
		Device:  "",
	}
}

func main() {
	cfg := config.New()
	err := cfg.Load()
	if err != nil {
		log.Fatal(err)
	}

	c := ui.NewCore(cfg)
	token, _ := keyring.Get(config.Name, "token")
	if token != "" {
		err = c.Run(token)
		if err != nil {
			log.Fatal(err)
		}

		c.Draw()
	} else {
		loginView := ui.NewLoginView(c)
		c.App.SetRoot(loginView, true)
	}

	err = c.App.Run()
	if err != nil {
		log.Fatal(err)
	}
}
