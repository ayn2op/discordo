package main

import (
	"flag"
	"log/slog"

	"github.com/ayn2op/discordo/cmd"
	"github.com/ayn2op/discordo/internal/constants"
	"github.com/zalando/go-keyring"
)

func main() {
	token := flag.String("token", "", "authentication token")
	flag.Parse()

	// If no token was provided, look it up in the keyring
	if *token == "" {
		t, err := keyring.Get(constants.Name, "token")
		if err != nil {
			slog.Info("failed to get token from keyring", "err", err)
		} else {
			*token = t
		}
	}

	if err := cmd.Run(*token); err != nil {
		slog.Error("failed to run", "err", err)
	}
}
