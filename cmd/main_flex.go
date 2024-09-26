package cmd

import (
	"log/slog"

	"github.com/ayn2op/discordo/internal/constants"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/zalando/go-keyring"
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
	case cfg.Keys.FocusGuildsTree:
		app.SetFocus(mf.guildsTree)
		return nil
	case cfg.Keys.FocusMessagesText:
		app.SetFocus(mf.messagesText)
		return nil
	case cfg.Keys.FocusMessageInput:
		app.SetFocus(mf.messageInput)
		return nil
	case cfg.Keys.Logout:
		app.Stop()

		if err := keyring.Delete(constants.Name, "token"); err != nil {
			slog.Error("failed to delete token from keyring", "err", err)
			return nil
		}

		return nil
	case cfg.Keys.ToggleGuildsTree:
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

	return event
}
