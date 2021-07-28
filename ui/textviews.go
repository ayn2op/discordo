package ui

import (
	"github.com/rivo/tview"
)

func NewMessagesTextView(app *tview.Application) (messagesTextView *tview.TextView) {
	messagesTextView = tview.NewTextView().
		SetDynamicColors(true).
		SetWrap(true).
		SetWordWrap(true).
		SetScrollable(true).
		ScrollToEnd().
		SetChangedFunc(onMessagesTextViewChanged(app))
	messagesTextView.
		SetBorder(true).
		SetBorderPadding(0, 0, 1, 1)

	return
}

func onMessagesTextViewChanged(app *tview.Application) func() {
	return func() {
		app.Draw()
	}
}
