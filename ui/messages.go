package ui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// NewMessagesView creates and returns a new messages textview.
func NewMessagesView(
	app *tview.Application,
	onMessagesTextViewInputCapture func(*tcell.EventKey) *tcell.EventKey,
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
		SetInputCapture(onMessagesTextViewInputCapture).
		SetBorder(true).
		SetBorderPadding(0, 0, 1, 0).
		SetTitleAlign(tview.AlignLeft)

	return v
}

// NewMessageInputField creates and returns a new message inputfield.
func NewMessageInputField(
	onMessageInputFieldInputCapture func(*tcell.EventKey) *tcell.EventKey,
) *tview.InputField {
	i := tview.NewInputField()
	i.
		SetPlaceholder("Message...").
		SetPlaceholderTextColor(tcell.ColorWhite).
		SetFieldBackgroundColor(tview.Styles.PrimitiveBackgroundColor).
		SetInputCapture(onMessageInputFieldInputCapture).
		SetBorder(true).
		SetBorderPadding(0, 0, 1, 0).
		SetTitleAlign(tview.AlignLeft)

	return i
}
