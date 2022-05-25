package main

import (
	"log"
	"os"

	"github.com/ayntgl/discordo/config"
	"github.com/ayntgl/discordo/core"
	"github.com/urfave/cli/v2"
	"github.com/zalando/go-keyring"
)

func main() {
	cliApp := cli.NewApp()
	cliApp.Name = config.Name
	cliApp.Usage = config.Usage

	token, _ := keyring.Get(config.Name, "token")
	cliApp.Flags = []cli.Flag{
		&cli.StringFlag{
			Name:    "token",
			Usage:   "The client authentication token.",
			Aliases: []string{"t"},
			Value:   token,
		},
	}

	cliApp.Action = func(ctx *cli.Context) error {
		cfg := config.New()
		err := cfg.Load()
		if err != nil {
			return err
		}

		c := core.New(ctx.String("token"), cfg)
		return c.Run()
	}

	err := cliApp.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
