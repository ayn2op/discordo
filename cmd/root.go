package cmd

import (
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"

	"github.com/ayn2op/arikawa/v3/utils/ws"
	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/discordo/internal/consts"
	"github.com/ayn2op/discordo/internal/logger"
	"github.com/ayn2op/discordo/internal/ui/root"
	"github.com/ayn2op/tview"
	"github.com/gdamore/tcell/v3"
)

func Run() error {
	configPath := flag.String("config-path", config.DefaultPath(), "path of the configuration file")
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

	cfg, err := config.Load(*configPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if cfg.DisableCache {
		// Never touch the log file at all, rather than creating then deleting
		// whatever path the user gave us, which could be something other than our own log file.
		slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: level})))

		// Remove attachments left behind by earlier runs, then whatever this run downloads.
		// Running two instances with disable_cache at once can race on this directory.
		os.RemoveAll(consts.AttachmentsCacheDir())       //nolint:errcheck
		defer os.RemoveAll(consts.AttachmentsCacheDir()) //nolint:errcheck
	} else {
		logFile, err := logger.Load(*logPath, level)
		if err != nil {
			return fmt.Errorf("failed to load logger: %w", err)
		}
		defer logFile.Close() //nolint:errcheck
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
