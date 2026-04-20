package login

import (
	"log/slog"

	"github.com/ayn2op/discordo/internal/clipboard"
	"github.com/ayn2op/tview"
)

type errMsg struct {
	err error
}

func setClipboard(content string) tview.Cmd {
	return func() tview.Msg {
		if err := clipboard.Write(clipboard.FmtText, []byte(content)); err != nil {
			slog.Error("failed to copy error message", "err", err)
			return nil
		}
		return nil
	}
}
