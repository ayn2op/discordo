package login

import (
	"context"
	"log/slog"

	"github.com/ayn2op/tview"
	"golang.design/x/clipboard"
)

type copyErrorMsg string

func setClipboard(content string) tview.Cmd {
	return func() tview.Msg {
		if _, err := clipboard.Write(context.Background(), clipboard.FmtText, []byte(content)); err != nil {
			slog.Error("failed to write to clipboard", "err", err)
			return nil
		}
		return nil
	}
}
