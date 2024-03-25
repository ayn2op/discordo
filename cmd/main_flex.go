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
	userList     *UserList

	guildsTreeVisible bool
	userListVisible bool
}

func newMainFlex() *MainFlex {
	mf := &MainFlex{
		Flex: tview.NewFlex(),

		guildsTree: newGuildsTree(),
		guildsTreeVisible: true,
		messagesText: newMessagesText(),
		messageInput: newMessageInput(),
		userList: newUserList(),
		userListVisible: true,
	}

	mf.init()
	mf.SetInputCapture(mf.onInputCapture)

	return mf
}

func (mf *MainFlex) init() {
	mf.Clear()

	if mf.guildsTreeVisible {
		mf.AddItem(mf.guildsTree, 0, 1, true)
	}

	chat := tview.NewFlex()
	chat.SetDirection(tview.FlexRow)
	chat.AddItem(mf.messagesText, 0, 1, false)
	chat.AddItem(mf.messageInput, 3, 1, false)
	mf.AddItem(chat, 0, 4, false)

	if mf.userListVisible {
		mf.AddItem(mf.userList, 0, 1, false)
	}
}

func (mf *MainFlex) onInputCapture(event *tcell.EventKey) *tcell.EventKey {
	switch event.Name() {
	case cfg.Keys.GuildsTree.Toggle:
		if mf.guildsTreeVisible {
			mf.RemoveItem(mf.guildsTree)
			app.SetFocus(mf)
			mf.guildsTreeVisible = false
		} else {
			mf.guildsTreeVisible = true
			mf.init()
			app.SetFocus(mf.guildsTree)
		}

		return nil
	case cfg.Keys.GuildsTree.Focus:
		if mf.guildsTreeVisible {
			app.SetFocus(mf.guildsTree)
		}
		return nil
	case cfg.Keys.MessagesText.Focus:
		app.SetFocus(mf.messagesText)
		return nil
	case cfg.Keys.MessageInput.Focus:
		app.SetFocus(mf.messageInput)
		return nil
	case cfg.Keys.UserList.Toggle:
		if mf.userListVisible {
			mf.RemoveItem(mf.userList)
			app.SetFocus(mf)
			mf.userListVisible = false
		} else {
			mf.userListVisible = true
			mf.init()
			app.SetFocus(mf.userList)
		}
		return nil
	case cfg.Keys.UserList.Focus:
		if mf.userListVisible {
			app.SetFocus(mf.userList)
		} else {
			mf.userListVisible = true
			mf.init()
			app.SetFocus(mf.userList)
		}

		return nil
	}

	return event
}
