package ui

import (
	"github.com/rivo/tview"
)

// NewMessagesTextView creates and returns a new messages textview.
func NewMessagesTextView(app *tview.Application) *tview.TextView {
	v := tview.NewTextView()
	v.
		SetRegions(true).
		SetDynamicColors(true).
		SetWordWrap(true).
		ScrollToEnd().
		SetChangedFunc(func() {
			app.Draw()
		}).
		SetBorder(true).
		SetBorderPadding(0, 0, 1, 0).
		SetTitleAlign(tview.AlignLeft)

	return v
}
