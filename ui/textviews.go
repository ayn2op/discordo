package ui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

var messagesTextViewBackgroundColor = tcell.GetColor("#1C1E26")

func NewMessagesTextView(onMessagesTextViewChanged func()) *tview.TextView {
	messagesTextView := tview.NewTextView().
		SetDynamicColors(true).
		SetWrap(true).
		SetWordWrap(true).
		SetScrollable(true).
		ScrollToEnd().
		SetChangedFunc(onMessagesTextViewChanged)
	messagesTextView.
		SetBorder(true).
		SetBorderPadding(0, 0, 1, 1).
		SetBackgroundColor(messagesTextViewBackgroundColor)

	return messagesTextView
}
