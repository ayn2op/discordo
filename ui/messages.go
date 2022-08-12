package ui

import "github.com/rivo/tview"

type MessagesPanel struct {
	*tview.TextView
}

func NewMessagesPanel() *MessagesPanel {
	tv := tview.NewTextView()

	tv.SetDynamicColors(true)
	tv.SetRegions(true)
	tv.SetWordWrap(true)

	tv.SetTitleAlign(tview.AlignLeft)
	tv.SetBorder(true)
	tv.SetBorderPadding(0, 0, 1, 1)

	return &MessagesPanel{
		TextView: tv,
	}
}

type MessageInput struct {
	*tview.InputField
}

func NewMessageInput() *MessageInput {
	i := tview.NewInputField()

	i.SetTitleAlign(tview.AlignLeft)
	i.SetBorder(true)
	i.SetBorderPadding(0, 0, 1, 1)

	return &MessageInput{
		InputField: i,
	}
}
