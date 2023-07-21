package main

import (
	"log"

	"github.com/alecthomas/kong"
	"github.com/ayn2op/discordo/cmd/run"
	"github.com/ayn2op/discordo/config"
	"github.com/zalando/go-keyring"
)

var cli struct {
	Run run.Cmd `cmd:"" default:"withargs"`
}

func main() {
	t, _ := keyring.Get(config.Name, "token")
	ctx := kong.Parse(&cli, kong.Vars{
		"token":      t,
		"configPath": config.DefaultPath(),
		"logPath":    config.DefaultLogPath(),
	})

	err := ctx.Run()
	if err != nil {
		log.Fatal(err)
	}
}
