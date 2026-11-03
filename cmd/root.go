package cmd

import (
	"flag"
	"fmt"
	"log/slog"

	"github.com/ayn2op/arikawa/v3/utils/ws"
	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/discordo/internal/logger"
	"github.com/ayn2op/discordo/internal/ui/root"
	"github.com/ayn2op/tview"
	"github.com/gdamore/tcell/v3"
)

func Run() error {
	configDir := flag.String("config-dir", config.DefaultDir(), "directory of the configuration file and themes")
	logPath := flag.String("log-path", logger.DefaultPath(), "path of the log file")
	logLevel := flag.String("log-level", "info", "log level")
	showVersion := flag.Bool("version", false, "print version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Printf("%s\n", buildVersion())
		return nil
	}

	var level slog.Level
	if err := level.UnmarshalText([]byte(*logLevel)); err != nil {
		return fmt.Errorf("invalid log level: %w", err)
	}
	ws.EnableRawEvents = level == slog.LevelDebug

	logFile, err := logger.Load(*logPath, level)
	if err != nil {
		return fmt.Errorf("failed to load logger: %w", err)
	}
	defer logFile.Close() //nolint:errcheck

	cfg, err := config.Load(*configDir)
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
	if cfg.Notifications.WhenUnfocused {
		screen.EnableFocus()
	}

	return tview.NewApplication(root.NewModel(cfg), tview.WithScreen(screen)).Run()
}
