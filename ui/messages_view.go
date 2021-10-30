package ui

import "github.com/rivo/tview"

func NewMessagesView() *tview.TextView {
	textView := tview.NewTextView()
	textView.
		SetRegions(true).
		SetDynamicColors(true).
		SetWordWrap(true).
		SetBorder(true).
		SetBorderPadding(0, 0, 1, 0).
		SetTitleAlign(tview.AlignLeft)

	return textView
}
