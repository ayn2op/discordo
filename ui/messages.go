package ui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// NewMessagesWidget creates and returns a new messages widget.
func NewMessagesWidget(
	app *tview.Application,
) *tview.TextView {
	w := tview.NewTextView()
	w.
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

	return w
}

// NewInputWidget creates and returns a new input widget.
func NewInputWidget() *tview.InputField {
	w := tview.NewInputField()
	w.
		SetPlaceholder("Message...").
		SetPlaceholderTextColor(tcell.ColorWhite).
		SetFieldBackgroundColor(tview.Styles.PrimitiveBackgroundColor).
		SetBorder(true).
		SetBorderPadding(0, 0, 1, 0).
		SetTitleAlign(tview.AlignLeft)

	return w
}
