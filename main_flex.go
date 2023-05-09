package main

import (
	"github.com/ayn2op/discordo/internal/config"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type MainFlex struct {
	*tview.Flex

	right        *tview.Flex
	guildsTree   *GuildsTree
	messagesText *MessagesText
	messageInput *MessageInput
}

func newMainFlex() *MainFlex {
	mf := &MainFlex{
		Flex: tview.NewFlex(),
		
		right:        tview.NewFlex(),
		guildsTree:   newGuildsTree(),
		messagesText: newMessagesText(),
		messageInput: newMessageInput(),
	}

	mf.right.SetDirection(tview.FlexRow)
	mf.right.AddItem(mf.messagesText, 0, 1, false)
	mf.right.AddItem(mf.messageInput, 3, 1, false)
	// The guilds tree is always focused first at start-up.
	mf.AddItem(mf.guildsTree, 0, 1, true)
	mf.AddItem(mf.right, 0, 4, false)
	mf.SetInputCapture(mf.onInputCapture)

	return mf
}

func (mf *MainFlex) onInputCapture(event *tcell.EventKey) *tcell.EventKey {
	switch event.Name() {
	case config.Current.Keys.GuildsTree.Toggle:
		if (mf.GetItemCount() > 1) {
			app.SetFocus(mf.guildsTree)
		}
		return nil
	case config.Current.Keys.GuildsTree.Focus:
		if (mf.GetItemCount() < 2) {
			mainFlex.RemoveItem(mf.right)
			mainFlex.AddItem(mf.guildsTree, 0, 1, true)
			mainFlex.AddItem(mf.right, 0, 4, false)
		} else {
			mainFlex.RemoveItem(mf.guildsTree)
			app.SetFocus(mf.messageInput)
		}
		return nil
			
	case config.Current.Keys.MessagesText.Focus:
		app.SetFocus(mf.messagesText)
		return nil
	case config.Current.Keys.MessageInput.Focus:
		app.SetFocus(mf.messageInput)
		return nil
	}

	return event
}
