package ui

import (
	"github.com/rivo/tview"
)

type View struct {
	*tview.Flex

	GuildsTree   *GuildsTree
	ChannelsTree *ChannelsTree
	MessagesText *MessagesText
	MessageInput *MessageInput

	app *Application
}

func newView(app *Application) *View {
	v := &View{
		Flex:         tview.NewFlex(),
		GuildsTree:   newGuildsTree(app),
		ChannelsTree: newChannelsTree(app),
		MessagesText: newMessagesText(app),
		MessageInput: newMessageInput(app),

		app: app,
	}

	left := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(v.GuildsTree, 10, 1, false).
		AddItem(v.ChannelsTree, 0, 1, false)
	right := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(v.MessagesText, 0, 1, false).
		AddItem(v.MessageInput, 3, 1, false)

	v.AddItem(left, 0, 1, false)
	v.AddItem(right, 0, 4, false)

	return v
}
