package ui

import (
	"log/slog"

	"github.com/ayn2op/tview"
)

type DialogButton struct {
	Label    string
	Cmd      tview.Cmd
	KeepOpen bool
}

type DialogMsg struct {
	Text    string
	Buttons []DialogButton
}

func ShowDialog(text string, buttons ...DialogButton) tview.Cmd {
	return func() tview.Msg { return DialogMsg{Text: text, Buttons: buttons} }
}

// ErrorDialog logs that what failed with err and returns the dialog that says so.
func ErrorDialog(what string, err error) DialogMsg {
	slog.Error("failed to "+what, "err", err)
	return DialogMsg{Text: "Failed to " + what + ": " + err.Error(), Buttons: []DialogButton{{Label: "OK"}}}
}
