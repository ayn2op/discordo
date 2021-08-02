package ui

import (
	"github.com/rivo/tview"
)

func NewApp() *tview.Application {
	app := tview.NewApplication().
		EnableMouse(true)

	return app
}
