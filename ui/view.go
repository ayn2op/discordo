package ui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type focusedId int

const (
	focusedIdGuildsView focusedId = iota
	focusedIdChannelsView
	focusedIdMessagesView
	focusedIdInputView
)

type View struct {
	*tview.Flex

	GuildsView   *GuildsView
	ChannelsView *ChannelsView
	MessagesView *MessagesView
	InputView    *InputView

	app     *Application
	focused focusedId
}

func newView(app *Application) *View {
	v := &View{
		Flex:         tview.NewFlex(),
		GuildsView:   newGuildsView(app),
		ChannelsView: newChannelsView(app),
		MessagesView: newMessagesView(app),
		InputView:    newInputView(app),

		app: app,
	}

	left := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(v.GuildsView, 10, 1, false).
		AddItem(v.ChannelsView, 0, 1, false)
	right := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(v.MessagesView, 0, 1, false).
		AddItem(v.InputView, 3, 1, false)

	v.AddItem(left, 0, 1, false)
	v.AddItem(right, 0, 4, false)

	v.SetInputCapture(v.onInputCapture)

	return v
}

func (v *View) onInputCapture(event *tcell.EventKey) *tcell.EventKey {
	switch event.Key() {
	case tcell.KeyEsc:
		v.focused = 0
	case tcell.KeyBacktab:
		// If the currently focused view is the guilds view (first), then focus the input view (last)
		if v.focused == 0 {
			v.focused = focusedIdInputView
		} else {
			v.focused--
		}

		v.setFocus()
	case tcell.KeyTab:
		// If the currently focused view is the input view (last), then focus the guilds view (first)
		if v.focused == focusedIdInputView {
			v.focused = focusedIdGuildsView
		} else {
			v.focused++
		}

		v.setFocus()
	}

	return event
}

func (v *View) setFocus() {
	var p tview.Primitive
	switch v.focused {
	case focusedIdGuildsView:
		p = v.GuildsView
	case focusedIdChannelsView:
		p = v.ChannelsView
	case focusedIdMessagesView:
		p = v.MessagesView
	case focusedIdInputView:
		p = v.InputView
	}

	v.app.SetFocus(p)
}
