package main

import (
	"log"

	"github.com/ayn2op/discordo/internal/root"
	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	p := tea.NewProgram(root.NewModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}
}
