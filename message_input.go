package main

import (
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

	mi.SetDoneFunc(mi.onDone)

	mi.SetFieldBackgroundColor(tview.Styles.PrimitiveBackgroundColor)
	mi.SetBorder(cfg.Theme.MessageInput.Border)
	padding := cfg.Theme.MessageInput.BorderPadding
	mi.SetBorderPadding(padding[0], padding[1], padding[2], padding[3])

	return mi
}

func (mi *MessageInput) onDone(key tcell.Key) {
	switch key {
	case tcell.KeyEnter:
		go mi.sendMessage()
	case tcell.KeyEscape:
		// Reset the message input.
		mi.SetText("")
	}
}

func (mi *MessageInput) sendMessage() error {
	text := mi.GetText()
	_, err := discordState.SendMessage(guildsTree.selectedChannel.ID, text)
	if err != nil {
		return err
	}

	// Reset the message input.
	mi.SetText("")
	return nil
}
