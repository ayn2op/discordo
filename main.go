package main

import (
	"log/slog"

	"github.com/ayn2op/discordo/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		slog.Error("failed to execute command", "err", err)
	}
}
