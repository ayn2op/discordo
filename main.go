package main

import (
	"os"

	"github.com/ayntgl/discordo/ui"
	"github.com/zalando/go-keyring"
)

func main() {
	app := ui.NewApp()
	app.Config.Load()

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
		ui.NewLoginForm(app, func() {
			ui.OnLoginFormLoginButtonSelected(app)
		}, false)

		app.SetRoot(app.LoginForm, true)
	}

	if err := app.EnableMouse(app.Config.General.Mouse).Run(); err != nil {
		panic(err)
	}
}
