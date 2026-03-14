package login

import (
	"log/slog"

	"github.com/ayn2op/discordo/internal/clipboard"
	"github.com/ayn2op/tview"
	"github.com/gdamore/tcell/v3"
)

func setClipboard(content string) tview.Command {
	return func() tcell.Event {
		if err := clipboard.Write(clipboard.FmtText, []byte(content)); err != nil {
			slog.Error("failed to copy error message", "err", err)
			return tcell.NewEventError(err)
		}
		return nil
	}
}
