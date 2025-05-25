package cmd

import (
	"flag"
	"log/slog"

	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/discordo/internal/consts"
	"github.com/ayn2op/discordo/internal/logger"
	"github.com/diamondburned/arikawa/v3/utils/ws"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/zalando/go-keyring"
)

var (
	discordState *State
	app          *App
)

func Run() error {
	logLevel := flag.String("log-level", "info", "log level")
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

	logFormat := flag.String("log-format", "text", "log format")
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

	token := flag.String("token", "", "authentication token")
	tok := *token
	if tok == "" {
		var err error
		tok, err = keyring.Get(consts.Name, "token")
		if err != nil {
			slog.Info("failed to retrieve token from keyring", "err", err)
		}
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	tview.Styles.PrimitiveBackgroundColor = tcell.GetColor(cfg.Theme.BackgroundColor)

	tview.Borders.Horizontal = cfg.Theme.Border.Preset.Horizontal
	tview.Borders.Vertical = cfg.Theme.Border.Preset.Vertical
	tview.Borders.TopLeft = cfg.Theme.Border.Preset.TopLeft
	tview.Borders.TopRight = cfg.Theme.Border.Preset.TopRight
	tview.Borders.BottomLeft = cfg.Theme.Border.Preset.BottomLeft
	tview.Borders.BottomRight = cfg.Theme.Border.Preset.BottomRight

	tview.Borders.HorizontalFocus = tview.Borders.Horizontal
	tview.Borders.VerticalFocus = tview.Borders.Vertical
	tview.Borders.TopLeftFocus = tview.Borders.TopLeft
	tview.Borders.TopRightFocus = tview.Borders.TopRight
	tview.Borders.BottomLeftFocus = tview.Borders.BottomLeft
	tview.Borders.BottomRightFocus = tview.Borders.BottomRight

	app = newApp(cfg)
	return app.run(tok)
}
