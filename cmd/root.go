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
