package ui

import (
	"github.com/rivo/tview"
)

// NewMessagesTextView creates and returns a new messages textview.
func NewMessagesTextView(app *tview.Application) (textV *tview.TextView) {
	textV = tview.NewTextView()
	textV.
		SetDynamicColors(true).
		SetWordWrap(true).
		ScrollToEnd().
		SetChangedFunc(func() {
			app.Draw()
		}).
		SetBorder(true).
		SetBorderPadding(0, 0, 1, 0).
		SetTitleAlign(tview.AlignLeft)

	return
}
