package ui

import (
	"context"
	"log/slog"

	"github.com/ayn2op/tview"
	"golang.design/x/clipboard"
)

// WriteClipboard copies text to the clipboard.
func WriteClipboard(text string) tview.Cmd {
	return func() tview.Msg {
		if _, err := clipboard.Write(context.Background(), clipboard.FmtText, []byte(text)); err != nil {
			slog.Error("failed to write to clipboard", "err", err)
		}
		return nil
	}
}
