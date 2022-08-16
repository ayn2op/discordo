package main

import (
	"log"
	"os"

	"github.com/ayntgl/discordo/config"
	"github.com/ayntgl/discordo/ui"
	"github.com/urfave/cli/v2"
)

const (
	name  = "discordo"
	usage = "A lightweight, secure, and feature-rich Discord terminal client"
)

func main() {
	cliApp := &cli.App{
		Name:                 name,
		Usage:                usage,
		EnableBashCompletion: true,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "token",
				Usage:   "The client authentication token.",
				Aliases: []string{"t"},
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
		var (
			c   *ui.Core
			err error
		)

		cfg := config.New()
		err = cfg.Load(ctx.String("config"))
		if err != nil {
			return err
		}

		token := ctx.String("token")
		if token != "" {
			c = ui.NewCore(token, cfg)
		}

		return c.Run()
	}

	err := cliApp.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
