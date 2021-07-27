package ui

import (
	"github.com/rivo/tview"
)

func NewApp() (app *tview.Application) {
	app = tview.NewApplication().
		EnableMouse(true)

	return
}
