package login

import (
	"context"
	"log/slog"

	"github.com/ayn2op/discordo/internal/ui"
	"github.com/ayn2op/tview"
	"golang.design/x/clipboard"
)

func setClipboard(content string) tview.Cmd {
	return func() tview.Msg {
		if _, err := clipboard.Write(context.Background(), clipboard.FmtText, []byte(content)); err != nil {
			slog.Error("failed to write to clipboard", "err", err)
		}
		return nil
	}
}

// selectTabMsg switches to the tab at an index.
type selectTabMsg int

func showErrorDialog(err error) tview.Cmd {
	slog.Error("failed to login", "err", err)
	message := err.Error()
	return ui.ShowModal(message,
		ui.ModalButton{Label: "Copy", Cmd: setClipboard(message), KeepOpen: true},
		ui.ModalButton{Label: "Close"},
	)
}
