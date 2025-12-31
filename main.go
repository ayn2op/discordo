package main

import (
	"log/slog"

	"github.com/ayn2op/discordo/cmd"
)

func main() {
	if err := cmd.Run(); err != nil {
		slog.Error("failed to run command", "err", err)
	}
}
