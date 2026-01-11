package cmd

import (
	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/discordo/internal/root"

	tea "charm.land/bubbletea/v2"
)

func Run() error {
	cfg, err := config.Load(config.DefaultPath())
	if err != nil {
		return err
	}

	p := tea.NewProgram(root.NewModel(cfg))
	_, err = p.Run()
	return err
}
