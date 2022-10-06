package ui

import (
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type MessagesView struct {
	*tview.TextView
	core            *Core
}

func newMessagesView(c *Core) *MessagesView {
	v := &MessagesView{
		TextView:        tview.NewTextView(),
		core:            c,
	}

	v.SetDynamicColors(true)
	v.SetRegions(true)
	v.SetWordWrap(true)
	v.SetInputCapture(v.onInputCapture)
	v.SetChangedFunc(func() {
		v.core.App.Draw()
	})

	v.SetTitle("Messages")
	v.SetTitleAlign(tview.AlignLeft)
	v.SetBorder(true)
	v.SetBorderPadding(0, 0, 1, 1)

	return v
}

func (v *MessagesView) onInputCapture(e *tcell.EventKey) *tcell.EventKey {
	if v.core.ChannelsView.selectedChannel == nil {
		return nil
	}

	// Messages should return messages ordered from latest to earliest.
	ms, err := v.core.State.Cabinet.Messages(v.core.ChannelsView.selectedChannel.ID)
	if err != nil || len(ms) == 0 {
		return nil
	}

	switch e.Name() {
	case v.core.Config.Keys.MessagesView.OpenActionsView:
		return v.openActionsView(ms)
	case v.core.Config.Keys.MessagesView.SelectPreviousMessage:
		return v.selectPreviousMessage(ms)
	case v.core.Config.Keys.MessagesView.SelectNextMessage:
		return v.selectNextMessage(ms)
	case v.core.Config.Keys.MessagesView.SelectFirstMessage:
		return v.selectFirstMessage(ms)
	case v.core.Config.Keys.MessagesView.SelectLastMessage:
		return v.selectLastMessage(ms)
	case "Esc":
		v.core.App.SetFocus(v.core.View)
		v.
			Clear().
			Highlight().
			SetTitle("")
		return nil
	}

	return e
}

func (v *MessagesView) selectPreviousMessage(ms []discord.Message) *tcell.EventKey {
	selInd := 0
	cSel := v.GetHighlights()
	// If there are no highlighted regions, select the latest (last) message.
	if len(cSel) == 0 {
		selInd = 0
	} else {
		cID, _ := discord.ParseSnowflake(cSel[0])
		selInd, _ = findMessageByID(ms, discord.MessageID(cID))
		// If the selected message is the oldest (first) message, select the latest (last) message.
		if selInd == len(ms)-1 {
			selInd = 0
		} else {
			selInd++
		}
	}

	v.Highlight(ms[selInd].ID.String())
	v.ScrollToHighlight()
	return nil
}

func (v *MessagesView) selectNextMessage(ms []discord.Message) *tcell.EventKey {
	selInd := 0
	cSel := v.GetHighlights()
	// If there are no highlighted regions, select the latest (last) message.
	if len(cSel) == 0 {
		selInd = 0
	} else {
		cID, _ := discord.ParseSnowflake(cSel[0])
		selInd, _ = findMessageByID(ms, discord.MessageID(cID))
		// If the selected message is the oldest (first) message, select the latest (last) message.
		if selInd == 0 {
			selInd = len(ms)-1
		} else {
			selInd--
		}
	}

	v.
		Highlight(ms[selInd].ID.String()).
		ScrollToHighlight()
	return nil
}

func (v *MessagesView) selectFirstMessage(ms []discord.Message) *tcell.EventKey {
	selID := len(ms) - 1
	v.
		Highlight(ms[selID].ID.String()).
		ScrollToHighlight()
	return nil
}

func (v *MessagesView) selectLastMessage(ms []discord.Message) *tcell.EventKey {
	selID := 0
	v.
		Highlight(ms[selID].ID.String()).
		ScrollToHighlight()
	return nil
}

func (v *MessagesView) openActionsView(ms []discord.Message) *tcell.EventKey {
	hs := v.GetHighlights()
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

	actionsView := newActionsView(v.core, m)
	v.core.App.SetRoot(actionsView, true)
	return nil
}
