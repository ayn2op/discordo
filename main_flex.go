package main

import (
	"log"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type MainFlex struct {
	*tview.Flex
}

func newMainFlex() *MainFlex {
	mf := &MainFlex{
		Flex: tview.NewFlex(),
	}

	// Initialize UI widgets
	guildsTree = newGuildsTree()
	messagesText = newMessagesText()
	messageInput = newMessageInput()

	right := tview.NewFlex()
	right.SetDirection(tview.FlexRow)
	right.AddItem(messagesText, 0, 1, false)
	right.AddItem(messageInput, 3, 1, false)

	// The guilds tree is always focused first at start-up.
	mf.AddItem(guildsTree, 0, 1, true)
	mf.AddItem(right, 0, 4, false)

	mf.SetInputCapture(mainFlex.onInputCapture)

	return mf
}

func (mf *MainFlex) onInputCapture(event *tcell.EventKey) *tcell.EventKey {
	log.Println(event.Name())
	switch event.Name() {
	case cfg.Keys.GuildsTree.Focus:
		app.SetFocus(guildsTree)
		return nil
	case cfg.Keys.MessagesText.Focus:
		app.SetFocus(messagesText)
		return nil
	case cfg.Keys.MessageInput.Focus:
		app.SetFocus(messageInput)
		return nil
	}

	return event
}
