package cmd

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type Mode uint

const (
	ModeNormal Mode = iota
	ModeInsert
)

type MainFlex struct {
	*tview.Flex

	mode         Mode
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

		mode:         ModeNormal,
		guildsTree: newGuildsTree(),
		guildsTreeVisible: true,
		messagesText: newMessagesText(),
		messageInput: newMessageInput(),
		userList: newUserList(),
		userListVisible: true,
	}

	app.SetBeforeDrawFunc(func(screen tcell.Screen) bool {
		switch mf.mode {
		case ModeNormal:
			mf.messageInput.SetBorderAttributes(tcell.AttrNone)
		case ModeInsert:
			mf.messageInput.SetBorderAttributes(tcell.AttrDim)
		}

		return false
	})

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
	switch mf.mode {
	case ModeNormal:
		switch event.Name() {
		case cfg.Keys.Normal.InsertMode:
			mf.mode = ModeInsert
			app.SetFocus(mf.messageInput)
			return nil

		case cfg.Keys.Normal.FocusGuildsTree:
			app.SetFocus(mf.guildsTree)
			return nil
		case cfg.Keys.Normal.FocusMessagesText:
			app.SetFocus(mf.messagesText)
			return nil
		case cfg.Keys.Normal.ToggleGuildsTree:
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
		}

		// do not propagate event to the children if the message input is focused in normal mode.
		if mf.messageInput.HasFocus() {
			return nil
		}
	case ModeInsert:
		switch event.Name() {
		case cfg.Keys.Insert.NormalMode:
			mf.mode = ModeNormal
			return nil
		}

		if !mf.messageInput.HasFocus() {
			return nil
		}
	}

	return event
}
