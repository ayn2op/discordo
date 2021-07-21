package ui

import (
	"github.com/rivo/tview"
)

func NewMessagesTextView(onMessagesTextViewChanged func()) (messagesTextView *tview.TextView) {
	messagesTextView = tview.NewTextView().
		SetDynamicColors(true).
		SetWrap(true).
		SetWordWrap(true).
		SetScrollable(true).
		ScrollToEnd().
		SetChangedFunc(onMessagesTextViewChanged)
	messagesTextView.
		SetBorder(true).
		SetBorderPadding(0, 0, 1, 1)

	return
}
