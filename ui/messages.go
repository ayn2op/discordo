package ui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// NewMessagesView creates and returns a new messages textview.
func NewMessagesView(
	app *tview.Application,
) *tview.TextView {
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

// NewMessageInputField creates and returns a new message inputfield.
func NewMessageInputField() *tview.InputField {
	i := tview.NewInputField()
	i.
		SetPlaceholder("Message...").
		SetPlaceholderTextColor(tcell.ColorWhite).
		SetFieldBackgroundColor(tview.Styles.PrimitiveBackgroundColor).
		SetBorder(true).
		SetBorderPadding(0, 0, 1, 0).
		SetTitleAlign(tview.AlignLeft)

	return i
}
