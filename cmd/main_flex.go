package cmd

import (
	"github.com/ayn2op/discordo/internal/config"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type Command func(event *tcell.EventKey) *tcell.EventKey

var commands = newCommands()

type Commands struct {
	common       map[string]Command
	guildsTree   map[string]Command
	messagesText map[string]Command
	messageInput map[string]Command
}

func newCommands() *Commands {
	return &Commands{
		common:       make(map[string]Command),
		guildsTree:   make(map[string]Command),
		messagesText: make(map[string]Command),
		messageInput: make(map[string]Command),
	}
}

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
}

func newMainFlex() *MainFlex {
	mf := &MainFlex{
		Flex: tview.NewFlex(),

		mode:         ModeNormal,
		guildsTree:   newGuildsTree(),
		messagesText: newMessagesText(),
		messageInput: newMessageInput(),
	}

	app.SetBeforeDrawFunc(func(screen tcell.Screen) bool {
		switch mf.mode {
		case ModeNormal:
			mf.messageInput.SetBorderColor(tcell.ColorPink)
		case ModeInsert:
			mf.messageInput.SetBorderColor(tcell.ColorRed)
		}

		return false
	})

	mf.init()
	mf.SetInputCapture(mf.onInputCapture)

	commands.common["normal_mode"] = func(event *tcell.EventKey) *tcell.EventKey {
		mf.mode = ModeNormal
		return nil
	}

	commands.common["insert_mode"] = func(event *tcell.EventKey) *tcell.EventKey {
		mf.mode = ModeInsert
		return nil
	}

	commands.common["focus_guilds_tree"] = func(event *tcell.EventKey) *tcell.EventKey {
		app.SetFocus(mf.guildsTree)
		return nil
	}

	commands.common["focus_messages_text"] = func(event *tcell.EventKey) *tcell.EventKey {
		app.SetFocus(mf.messagesText)
		return nil
	}

	commands.common["focus_message_input"] = func(event *tcell.EventKey) *tcell.EventKey {
		app.SetFocus(mf.messageInput)
		return nil
	}

	commands.common["toggle_guilds_tree"] = func(event *tcell.EventKey) *tcell.EventKey {
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

	commands.common["select_down"] = func(event *tcell.EventKey) *tcell.EventKey {
		return event
	}

	commands.common["select_up"] = func(event *tcell.EventKey) *tcell.EventKey {
		return event
	}

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

func (mf *MainFlex) keysByMode() *config.Keys {
	switch mainFlex.mode {
	case ModeNormal:
		return &cfg.Keys.Normal
	case ModeInsert:
		return &cfg.Keys.Insert
	}

	return nil
}

func (mf *MainFlex) onInputCapture(event *tcell.EventKey) *tcell.EventKey {
	if commandName, ok := mf.keysByMode().Common[event.Name()]; ok {
		if command, ok := commands.common[commandName]; ok {
			return command(event)
		}
	}

	return event
}
