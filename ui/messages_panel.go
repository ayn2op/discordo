package ui

import (
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type MessagesPanel struct {
	*tview.TextView
	// The index of the currently selected message. A negative index indicates that there is no currently selected message.
	SelectedMessage int

	core *Core
}

func NewMessagesPanel(c *Core) *MessagesPanel {
	mp := &MessagesPanel{
		TextView:        tview.NewTextView(),
		SelectedMessage: -1,

		core: c,
	}

	mp.SetDynamicColors(true)
	mp.SetRegions(true)
	mp.SetWordWrap(true)
	mp.SetInputCapture(mp.onInputCapture)
	mp.SetChangedFunc(func() {
		mp.core.App.Draw()
	})

	mp.SetTitle("Messages")
	mp.SetTitleAlign(tview.AlignLeft)
	mp.SetBorder(true)
	mp.SetBorderPadding(0, 0, 1, 1)

	return mp
}

func (mp *MessagesPanel) onInputCapture(e *tcell.EventKey) *tcell.EventKey {
	if mp.core.ChannelsTree.SelectedChannel == nil {
		return nil
	}

	// Messages should return messages ordered from latest to earliest.
	ms, err := mp.core.State.Cabinet.Messages(mp.core.ChannelsTree.SelectedChannel.ID)
	if err != nil || len(ms) == 0 {
		return nil
	}

	// Defaults
	switch e.Name() {
	case mp.core.Config.Keys.MessagesPanel.OpenActionsList:
		return mp.openMessageActionsList(ms)

	case mp.core.Config.Keys.MessagesPanel.SelectPreviousMessage:
		return mp.selectPreviousMessage(ms)
	case mp.core.Config.Keys.MessagesPanel.SelectNextMessage:
		return mp.selectNextMessage(ms)
	case mp.core.Config.Keys.MessagesPanel.SelectFirstMessage:
		return mp.selectFirstMessage(ms)
	case mp.core.Config.Keys.MessagesPanel.SelectLastMessage:
		return mp.selectLastMessage(ms)
	case "Esc":
		mp.SelectedMessage = -1
		mp.core.App.SetFocus(mp.core.View)
		mp.
			Clear().
			Highlight().
			SetTitle("")
		return nil
	}

	return e
}

func (mp *MessagesPanel) selectPreviousMessage(ms []discord.Message) *tcell.EventKey {
	// If there are no highlighted regions, select the latest (last) message in the messages panel.
	if len(mp.GetHighlights()) == 0 {
		mp.SelectedMessage = 0
	} else {
		// If the selected message is the oldest (first) message, select the latest (last) message in the messages panel.
		if mp.SelectedMessage == len(ms)-1 {
			mp.SelectedMessage = 0
		} else {
			mp.SelectedMessage++
		}
	}

	mp.Highlight(ms[mp.SelectedMessage].ID.String())
	mp.ScrollToHighlight()
	return nil
}

func (mp *MessagesPanel) selectNextMessage(ms []discord.Message) *tcell.EventKey {
	// If there are no highlighted regions, select the latest (last) message in the messages panel.
	if len(mp.GetHighlights()) == 0 {
		mp.SelectedMessage = 0
	} else {
		// If the selected message is the latest (last) message, select the oldest (first) message in the messages panel.
		if mp.SelectedMessage == 0 {
			mp.SelectedMessage = len(ms) - 1
		} else {
			mp.SelectedMessage--
		}
	}

	mp.
		Highlight(ms[mp.SelectedMessage].ID.String()).
		ScrollToHighlight()
	return nil
}

func (mp *MessagesPanel) selectFirstMessage(ms []discord.Message) *tcell.EventKey {
	mp.SelectedMessage = len(ms) - 1
	mp.
		Highlight(ms[mp.SelectedMessage].ID.String()).
		ScrollToHighlight()
	return nil
}

func (mp *MessagesPanel) selectLastMessage(ms []discord.Message) *tcell.EventKey {
	mp.SelectedMessage = 0
	mp.
		Highlight(ms[mp.SelectedMessage].ID.String()).
		ScrollToHighlight()
	return nil
}

func (mp *MessagesPanel) openMessageActionsList(ms []discord.Message) *tcell.EventKey {
	hs := mp.GetHighlights()
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

	actionsList := NewMessageActionsList(mp.core, m)
	mp.core.App.SetRoot(actionsList, true)
	return nil
}
