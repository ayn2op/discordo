package main

import (
	"log"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type MessageInput struct {
	*tview.InputField
}

func newMessageInput() *MessageInput {
	mi := &MessageInput{
		InputField: tview.NewInputField(),
	}

	mi.SetFieldBackgroundColor(tview.Styles.PrimitiveBackgroundColor)
	mi.SetInputCapture(mi.onInputCapture)

	mi.SetTitleAlign(tview.AlignLeft)

	padding := cfg.Theme.MessageInput.BorderPadding
	mi.SetBorder(cfg.Theme.MessageInput.Border)
	mi.SetBorderPadding(padding[0], padding[1], padding[2], padding[3])

	return mi
}

func (mi *MessageInput) onInputCapture(event *tcell.EventKey) *tcell.EventKey {
	switch event.Name() {
	case cfg.Keys.MessageInput.Send:
		mi.sendAction()
		return nil
	}

	return event
}

func (mi *MessageInput) sendAction() {
	text := mi.GetText()
	_, err := discordState.SendMessage(guildsTree.selectedChannel.ID, text)
	if err != nil {
		log.Println(err)
		return
	}

	// Reset the message input.
	mi.SetText("")
}
