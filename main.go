package main

import (
	"flag"

	"github.com/ayn2op/discordo/cmd"
	"github.com/ayn2op/discordo/internal/config"
	"github.com/charmbracelet/log"
	"github.com/zalando/go-keyring"
)

func main() {
	token := flag.String("token", "", "authentication token")
	flag.Parse()

	// If no token was provided, look it up in the keyring
	if *token == "" {
		t, err := keyring.Get(config.Name, "token")
		if err != nil {
			log.Info("failed to get token from keyring", "err", err)
		} else {
			*token = t
		}
	}

	if err := cmd.Run(*token); err != nil {
		log.Fatal(err)
	}
}
