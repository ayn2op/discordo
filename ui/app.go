package ui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func NewApp() (app *tview.Application) {
	app = tview.NewApplication().
		EnableMouse(true).
		SetInputCapture(onAppInputCapture)

	return
}

func onAppInputCapture(event *tcell.EventKey) *tcell.EventKey {
	return event
}
