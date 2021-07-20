package ui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func NewApplication(onApplicationInputCapture func(event *tcell.EventKey) *tcell.EventKey) *tview.Application {
	app := tview.NewApplication().
		EnableMouse(true).
		SetInputCapture(onApplicationInputCapture)

	return app
}
