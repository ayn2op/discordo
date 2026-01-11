package cmd

import (
	"flag"

	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/discordo/internal/logger"
	"github.com/ayn2op/discordo/internal/root"

	tea "charm.land/bubbletea/v2"
)

func Run() error {
	var (
		token      string
		configPath string
		logPath    string
		logLevel   string
	)
	flag.StringVar(&token, "token", "", "authentication token (default: $DISCORDO_TOKEN or keyring)")
	flag.StringVar(&configPath, "config-path", config.DefaultPath(), "path of the configuration file")
	flag.StringVar(&logPath, "log-path", logger.DefaultPath(), "path of the log file")
	flag.StringVar(&logLevel, "log-level", "info", "log level")
	flag.Parse()

	cfg, err := config.Load(config.DefaultPath())
	if err != nil {
		return err
	}

	p := tea.NewProgram(root.NewModel(cfg))
	_, err = p.Run()
	return err
}
