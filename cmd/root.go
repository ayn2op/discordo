package cmd

import (
	"flag"
	"log/slog"

	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/discordo/internal/consts"
	"github.com/ayn2op/discordo/internal/logger"
	"github.com/ayn2op/tview"
	"github.com/diamondburned/arikawa/v3/utils/ws"
	"github.com/gdamore/tcell/v2"
	"github.com/zalando/go-keyring"
)

var (
	discordState *state
	app          *application
)

func Run() error {
	logLevel := flag.String("log-level", "info", "log level")
	logFormat := flag.String("log-format", "text", "log format")
	token := flag.String("token", "", "authentication token")
	configPath := flag.String("config", config.DefaultPath(), "path to the configuration file")
	flag.Parse()

	var level slog.Level
	switch *logLevel {
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

	var format logger.Format
	switch *logFormat {
	case "text":
		format = logger.FormatText
	case "json":
		format = logger.FormatJson
	}

	if err := logger.Load(format, level); err != nil {
		return err
	}

	tok := *token
	if tok == "" {
		var err error
		tok, err = keyring.Get(consts.Name, "token")
		if err != nil {
			slog.Info("failed to retrieve token from keyring", "err", err)
		}
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		return err
	}

	tview.Styles.PrimitiveBackgroundColor = tcell.GetColor(cfg.Theme.BackgroundColor)

	tview.BordersSet.Horizontal = cfg.Theme.Border.Preset.Horizontal
	tview.BordersSet.Vertical = cfg.Theme.Border.Preset.Vertical
	tview.BordersSet.TopLeft = cfg.Theme.Border.Preset.TopLeft
	tview.BordersSet.TopRight = cfg.Theme.Border.Preset.TopRight
	tview.BordersSet.BottomLeft = cfg.Theme.Border.Preset.BottomLeft
	tview.BordersSet.BottomRight = cfg.Theme.Border.Preset.BottomRight

	tview.BordersSet.HorizontalFocus = tview.BordersSet.Horizontal
	tview.BordersSet.VerticalFocus = tview.BordersSet.Vertical
	tview.BordersSet.TopLeftFocus = tview.BordersSet.TopLeft
	tview.BordersSet.TopRightFocus = tview.BordersSet.TopRight
	tview.BordersSet.BottomLeftFocus = tview.BordersSet.BottomLeft
	tview.BordersSet.BottomRightFocus = tview.BordersSet.BottomRight

	app = newApp(cfg)
	return app.run(tok)
}
