package main

import (
	"log"
	"os"

	"github.com/ayntgl/discordo/config"
	"github.com/ayntgl/discordo/ui"
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
		cfg := config.New()
		err := cfg.Load(ctx.String("config"))
		if err != nil {
			return err
		}

		token := ctx.String("token")
		c := ui.NewCore(token, cfg)
		if token != "" {
			c.DrawFlex()
			c.Application.SetFocus(c.GuildsList)
		} else {
			lf := ui.NewLoginForm()

			lf.AddButton("Login", func() {
				email := lf.GetFormItemByLabel(ui.EmailInputFieldLabel).(*tview.InputField).GetText()
				password := lf.GetFormItemByLabel(ui.PasswordInputFieldLabel).(*tview.InputField).GetText()
				if email == "" || password == "" {
					return
				}

				// Login using email and password
				lr, err := c.State.Login(email, password)
				if err != nil {
					log.Fatal(err)
				}

				if lr.Token != "" && !lr.MFA {
					c.State.Token = token

					c.Application.SetFocus(c.GuildsList)
					go keyring.Set(name, "token", lr.Token)
				}
			})

			c.Application.SetRoot(lf, true)
		}

		return c.Application.Run()
	}

	err := cliApp.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
