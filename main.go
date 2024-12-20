package main

import (
	"flag"
	"log/slog"

	"github.com/ayn2op/discordo/cmd"
)

func main() {
	token := flag.String("token", "", "authentication token")
	flag.Parse()

	if err := cmd.Run(*token); err != nil {
		slog.Error("failed to run", "err", err)
	}
}
