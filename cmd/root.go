package cmd

import (
	"log/slog"

	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/discordo/internal/consts"
	"github.com/ayn2op/discordo/internal/logger"
	"github.com/rivo/tview"
	"github.com/spf13/cobra"
	"github.com/zalando/go-keyring"
)

var (
	discordState *State
	app          *App
)

var (
	rootCmd = &cobra.Command{
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := logger.Load(); err != nil {
				return err
			}

			token, _ := cmd.Flags().GetString("token")
			if token == "" {
				var err error
				token, err = keyring.Get(consts.Name, "token")
				if err != nil {
					slog.Info("failed to retrieve token from keyring", "err", err)
				}
			}

			cfg, err := config.Load()
			if err != nil {
				return err
			}

			tview.Borders.Horizontal = cfg.Theme.Border.Type.Horizontal
			tview.Borders.Vertical = cfg.Theme.Border.Type.Vertical
			tview.Borders.TopLeft = cfg.Theme.Border.Type.TopLeft
			tview.Borders.TopRight = cfg.Theme.Border.Type.TopRight
			tview.Borders.BottomLeft = cfg.Theme.Border.Type.BottomLeft
			tview.Borders.BottomRight = cfg.Theme.Border.Type.BottomRight

			tview.Borders.HorizontalFocus = tview.Borders.Horizontal
			tview.Borders.VerticalFocus = tview.Borders.Vertical
			tview.Borders.TopLeftFocus = tview.Borders.TopLeft
			tview.Borders.TopRightFocus = tview.Borders.TopRight
			tview.Borders.BottomLeftFocus = tview.Borders.BottomLeft
			tview.Borders.BottomRightFocus = tview.Borders.BottomRight

			app = newApp(cfg)
			return app.run(token)
		},
	}

	Execute = rootCmd.Execute
)

func init() {
	rootCmd.Flags().StringP("token", "t", "", "the authentication token")
}
