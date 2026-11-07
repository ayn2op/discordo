package ui

import (
	"context"
	"log/slog"

	"github.com/ayn2op/tview"
	"github.com/ayn2op/tview/cmd"
	"golang.design/x/clipboard"
)

// WriteClipboard copies text to the clipboard.
func WriteClipboard(text string) tview.Cmd {
	return cmd.Task(func() tview.Msg {
		if _, err := clipboard.Write(context.Background(), clipboard.FmtText, []byte(text)); err != nil {
			slog.Error("failed to write to clipboard", "err", err)
		}
		return nil
	})
}
