package main

import (
	"flag"
	"log"

	"github.com/ayn2op/discordo/cmd/run"
	"github.com/ayn2op/discordo/config"
	"github.com/zalando/go-keyring"
)

func main() {
	t, _ := keyring.Get(config.Name, "token")
	token := flag.String("token", t, "The authentication token.")
	flag.Parse()

	if err := run.Run(*token); err != nil {
		log.Fatal(err)
	}
}
