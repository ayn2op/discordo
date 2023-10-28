package cmd

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type MainFlex struct {
	*tview.Flex

	guildsTree   *GuildsTree
	messagesText *MessagesText
	messageInput *MessageInput
}

func newMainFlex() *MainFlex {
	mf := &MainFlex{
		Flex: tview.NewFlex(),

		guildsTree:   newGuildsTree(),
		messagesText: newMessagesText(),
		messageInput: newMessageInput(),
	}

	mf.init()
	mf.SetInputCapture(mf.onInputCapture)

	return mf
}

func (mf *MainFlex) init() {
	mf.Clear()

	right := tview.NewFlex()
	right.SetDirection(tview.FlexRow)
	right.AddItem(mf.messagesText, 0, 1, false)
	right.AddItem(mf.messageInput, 3, 1, false)
	// The guilds tree is always focused first at start-up.
	mf.AddItem(mf.guildsTree, 0, 1, true)
	mf.AddItem(right, 0, 4, false)
}

func (mf *MainFlex) onInputCapture(event *tcell.EventKey) *tcell.EventKey {
	switch event.Name() {
	case cfg.Keys.GuildsTree.Toggle:
		// The guilds tree is visible if the numbers of items is two.
		if mf.GetItemCount() == 2 {
			mf.RemoveItem(mf.guildsTree)

			if mf.guildsTree.HasFocus() {
				app.SetFocus(mf)
			}
		} else {
			mf.init()
			app.SetFocus(mf.guildsTree)
		}

		return nil
	case cfg.Keys.GuildsTree.Focus:
		app.SetFocus(mf.guildsTree)
		return nil
	case cfg.Keys.MessagesText.Focus:
		app.SetFocus(mf.messagesText)
		return nil
	case cfg.Keys.MessageInput.Focus:
		app.SetFocus(mf.messageInput)
		return nil
	}

	return event
}
