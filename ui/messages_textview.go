package ui

import "github.com/rivo/tview"

func NewMessagesTextView() *tview.TextView {
	v := tview.NewTextView()
	v.
		SetRegions(true).
		SetDynamicColors(true).
		SetWordWrap(true).
		SetTitleAlign(tview.AlignLeft).
		SetBorder(true).
		SetBorderPadding(0, 0, 1, 0)

	return v
}
