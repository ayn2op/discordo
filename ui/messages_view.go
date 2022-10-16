package ui

import (
	"log"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type MessagesView struct {
	*tview.TextView
	// The index of the currently selected message. A negative index indicates that there is no currently selected message.
	selectedMessage int
	app             *Application
}

func newMessagesView(app *Application) *MessagesView {
	v := &MessagesView{
		TextView:        tview.NewTextView(),
		selectedMessage: -1,
		app:             app,
	}

	v.SetDynamicColors(true)
	v.SetRegions(true)
	v.SetWordWrap(true)
	v.SetInputCapture(v.onInputCapture)
	v.SetChangedFunc(func() {
		v.app.Draw()
	})

	v.SetTitle("Messages")
	v.SetTitleAlign(tview.AlignLeft)
	v.SetBorder(true)
	v.SetBorderPadding(0, 0, 1, 1)

	return v
}

func (v *MessagesView) setTitle(c *discord.Channel) {
	title := channelToString(*c)
	if c.Topic != "" {
		title += " - " + parseMarkdown(c.Topic)
	}

	v.SetTitle(title)
}

func (v *MessagesView) loadMessages(c *discord.Channel) {
	// The returned slice will be sorted from latest to oldest.
	ms, err := v.app.state.Messages(c.ID, v.app.config.MessagesLimit)
	if err != nil {
		log.Println(err)
		return
	}

	for i := len(ms) - 1; i >= 0; i-- {
		_, err = v.app.view.MessagesView.Write(buildMessage(v.app, ms[i]))
		if err != nil {
			log.Println(err)
			continue
		}
	}

	v.ScrollToEnd()
}

func (v *MessagesView) onInputCapture(e *tcell.EventKey) *tcell.EventKey {
	if v.app.view.ChannelsView.selected == nil {
		return nil
	}

	// Messages should return messages ordered from latest to earliest.
	ms, err := v.app.state.Cabinet.Messages(v.app.view.ChannelsView.selected.ID)
	if err != nil || len(ms) == 0 {
		return nil
	}

	switch e.Name() {
	case v.app.config.Keys.MessagesView.OpenActionsView:
		return v.openActionsView(ms)

	case v.app.config.Keys.MessagesView.SelectPreviousMessage:
		return v.selectPreviousMessage(ms)
	case v.app.config.Keys.MessagesView.SelectNextMessage:
		return v.selectNextMessage(ms)
	case v.app.config.Keys.MessagesView.SelectFirstMessage:
		return v.selectFirstMessage(ms)
	case v.app.config.Keys.MessagesView.SelectLastMessage:
		return v.selectLastMessage(ms)
	case "Esc":
		v.selectedMessage = -1
		v.app.SetFocus(v.app.view)
		v.
			Clear().
			Highlight().
			SetTitle("")
		return nil
	}

	return e
}

func (v *MessagesView) selectPreviousMessage(ms []discord.Message) *tcell.EventKey {
	// If there are no highlighted regions, select the latest (last) message.
	if len(v.GetHighlights()) == 0 {
		v.selectedMessage = 0
	} else {
		// If the selected message is the oldest (first) message, select the latest (last) message.
		if v.selectedMessage == len(ms)-1 {
			v.selectedMessage = 0
		} else {
			v.selectedMessage++
		}
	}

	v.Highlight(ms[v.selectedMessage].ID.String())
	v.ScrollToHighlight()
	return nil
}

func (v *MessagesView) selectNextMessage(ms []discord.Message) *tcell.EventKey {
	// If there are no highlighted regions, select the latest (last) message.
	if len(v.GetHighlights()) == 0 {
		v.selectedMessage = 0
	} else {
		// If the selected message is the latest (last) message, select the oldest (first) message.
		if v.selectedMessage == 0 {
			v.selectedMessage = len(ms) - 1
		} else {
			v.selectedMessage--
		}
	}

	v.
		Highlight(ms[v.selectedMessage].ID.String()).
		ScrollToHighlight()
	return nil
}

func (v *MessagesView) selectFirstMessage(ms []discord.Message) *tcell.EventKey {
	v.selectedMessage = len(ms) - 1
	v.
		Highlight(ms[v.selectedMessage].ID.String()).
		ScrollToHighlight()
	return nil
}

func (v *MessagesView) selectLastMessage(ms []discord.Message) *tcell.EventKey {
	v.selectedMessage = 0
	v.
		Highlight(ms[v.selectedMessage].ID.String()).
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

	actionsView := newActionsView(v.app, m)
	v.app.SetRoot(actionsView, true)
	return nil
}
