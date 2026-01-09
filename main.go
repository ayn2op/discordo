package main

import (
	"log/slog"
	"os"

	tea "charm.land/bubbletea/v2"
	"github.com/ayn2op/discordo/internal/root"
)

func main() {
	p := tea.NewProgram(root.NewModel())
	if _, err := p.Run(); err != nil {
		slog.Error("failed to run program", "err", err)
		os.Exit(1)
	}
}
