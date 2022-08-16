package ui

import "github.com/rivo/tview"

type MessagesPanel struct {
	*tview.TextView
	*Core
}

func NewMessagesPanel(c *Core) *MessagesPanel {
	mp := &MessagesPanel{
		TextView: tview.NewTextView(),
		Core:     c,
	}

	mp.SetDynamicColors(true)
	mp.SetRegions(true)
	mp.SetWordWrap(true)

	mp.SetTitleAlign(tview.AlignLeft)
	mp.SetBorder(true)
	mp.SetBorderPadding(0, 0, 1, 1)

	return mp
}

type MessageInput struct {
	*tview.InputField
	*Core
}

func NewMessageInput(c *Core) *MessageInput {
	mi := &MessageInput{
		InputField: tview.NewInputField(),
	}

	mi.SetTitleAlign(tview.AlignLeft)
	mi.SetBorder(true)
	mi.SetBorderPadding(0, 0, 1, 1)

	return mi
}
