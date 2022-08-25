package ui

import (
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type MessagesPanel struct {
	*tview.TextView
	app *App
	// The index of the currently selected message.
	SelectedMessage int
}

func NewMessagesPanel(app *App) *MessagesPanel {
	mp := &MessagesPanel{
		TextView: tview.NewTextView(),
		app:      app,
		// Negative index indicates that there is no currently selected message.
		SelectedMessage: -1,
	}

	mp.SetDynamicColors(true)
	mp.SetRegions(true)
	mp.SetWordWrap(true)
	mp.SetInputCapture(mp.onInputCapture)
	mp.SetChangedFunc(func() {
		mp.app.Draw()
	})

	mp.SetTitle("Messages")
	mp.SetTitleAlign(tview.AlignLeft)
	mp.SetBorder(true)
	mp.SetBorderPadding(0, 0, 1, 1)

	return mp
}

func (mp *MessagesPanel) onInputCapture(e *tcell.EventKey) *tcell.EventKey {
	if mp.app.ChannelsTree.SelectedChannel == nil {
		return nil
	}

	// Messages should return messages ordered from latest to earliest.
	ms, err := mp.app.State.Cabinet.Messages(mp.app.ChannelsTree.SelectedChannel.ID)
	if err != nil || len(ms) == 0 {
		return nil
	}

	switch e.Name() {
	case mp.app.Config.Keys.SelectPreviousMessage:
		// If there are no highlighted regions, select the latest (last) message in the messages panel.
		if len(mp.app.MessagesPanel.GetHighlights()) == 0 {
			mp.SelectedMessage = 0
		} else {
			// If the selected message is the oldest (first) message, select the latest (last) message in the messages panel.
			if mp.SelectedMessage == len(ms)-1 {
				mp.SelectedMessage = 0
			} else {
				mp.SelectedMessage++
			}
		}

		mp.app.MessagesPanel.
			Highlight(ms[mp.SelectedMessage].ID.String()).
			ScrollToHighlight()
		return nil
	case mp.app.Config.Keys.SelectNextMessage:
		// If there are no highlighted regions, select the latest (last) message in the messages panel.
		if len(mp.app.MessagesPanel.GetHighlights()) == 0 {
			mp.SelectedMessage = 0
		} else {
			// If the selected message is the latest (last) message, select the oldest (first) message in the messages panel.
			if mp.SelectedMessage == 0 {
				mp.SelectedMessage = len(ms) - 1
			} else {
				mp.SelectedMessage--
			}
		}

		mp.app.MessagesPanel.
			Highlight(ms[mp.SelectedMessage].ID.String()).
			ScrollToHighlight()
		return nil
	case mp.app.Config.Keys.SelectFirstMessage:
		mp.SelectedMessage = len(ms) - 1
		mp.app.MessagesPanel.
			Highlight(ms[mp.SelectedMessage].ID.String()).
			ScrollToHighlight()
		return nil
	case mp.app.Config.Keys.SelectLastMessage:
		mp.SelectedMessage = 0
		mp.app.MessagesPanel.
			Highlight(ms[mp.SelectedMessage].ID.String()).
			ScrollToHighlight()
		return nil
	case mp.app.Config.Keys.OpenMessageActionsList:
		hs := mp.app.MessagesPanel.GetHighlights()
		if len(hs) == 0 {
			return nil
		}

		mID, err := discord.ParseSnowflake(hs[0])
		if err != nil {
			return nil
		}

		_, m := findMessageByID(ms, discord.MessageID(mID))
		if m == nil {
			return nil
		}

		actionsList := NewMessageActionsList(mp.app, m)
		mp.app.SetRoot(actionsList, true)
		return nil
	case "Esc":
		mp.SelectedMessage = -1
		mp.app.SetFocus(mp.app.MainFlex)
		mp.app.MessagesPanel.
			Clear().
			Highlight().
			SetTitle("")
		return nil
	}

	return e
}
