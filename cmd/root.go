package cmd

import (
	"flag"
	"fmt"
	"log/slog"

	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/discordo/internal/logger"
	"github.com/ayn2op/discordo/internal/ui/root"
	"github.com/ayn2op/tview"
	"github.com/diamondburned/arikawa/v3/utils/ws"
	"github.com/gdamore/tcell/v3"
)

var (
	configPath string
	logPath    string
	logLevel   string
)

func Run() error {
	flag.StringVar(&configPath, "config-path", config.DefaultPath(), "path of the configuration file")
	flag.StringVar(&logPath, "log-path", logger.DefaultPath(), "path of the log file")
	flag.StringVar(&logLevel, "log-level", "info", "log level")
	flag.Parse()

	var level slog.Level
	switch logLevel {
	case "debug":
		ws.EnableRawEvents = true
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	}

	if err := logger.Load(logPath, level); err != nil {
		return fmt.Errorf("failed to load logger: %w", err)
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	screen, err := tcell.NewScreen()
	if err != nil {
		return fmt.Errorf("failed to create screen: %w", err)
	}

	if err := screen.Init(); err != nil {
		return fmt.Errorf("failed to init screen: %w", err)
	}

	if cfg.Mouse {
		screen.EnableMouse()
	}
	screen.EnablePaste()
	screen.EnableFocus()

	tview.Styles = tview.Theme{}
	app := tview.NewApplication()
	app.SetRoot(root.NewView(cfg, app))
	app.SetScreen(screen)
	return app.Run()
}
