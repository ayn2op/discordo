package login

import (
	"log/slog"

	"github.com/ayn2op/discordo/internal/ui"
	"github.com/ayn2op/tview"
)

// selectTabMsg switches to the tab at an index.
type selectTabMsg int

func showErrorDialog(err error) tview.Cmd {
	slog.Error("failed to login", "err", err)
	message := err.Error()
	return ui.ShowDialog(message,
		ui.DialogButton{Label: "Copy", Cmd: ui.WriteClipboard(message), KeepOpen: true},
		ui.DialogButton{Label: "Close"},
	)
}
