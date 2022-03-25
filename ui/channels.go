package ui

import (
	"github.com/ayntgl/astatine"
	"github.com/ayntgl/discordo/discord"
	"github.com/rivo/tview"
)

type ChannelsTreeView struct {
	*tview.TreeView
	app *App
}

func NewChannelsTreeView(app *App) *ChannelsTreeView {
	ctv := &ChannelsTreeView{
		TreeView: tview.NewTreeView(),
		app:      app,
	}

	ctv.SetTopLevel(1)
	ctv.SetRoot(tview.NewTreeNode(""))
	ctv.SetTitle("Channels")
	ctv.SetTitleAlign(tview.AlignLeft)
	ctv.SetBorder(true)
	ctv.SetBorderPadding(0, 0, 1, 1)
	ctv.SetSelectedFunc(ctv.onSelected)
	return ctv
}

func (ctv *ChannelsTreeView) onSelected(n *tview.TreeNode) {
	ctv.app.SelectedMessage = -1
	ctv.app.MessagesTextView.
		Highlight().
		Clear().
		SetTitle("")
	ctv.app.MessageInputField.SetText("")

	c, err := ctv.app.Session.State.Channel(n.GetReference().(string))
	if err != nil {
		return
	}

	if c.Type == astatine.ChannelTypeGuildCategory {
		n.SetExpanded(!n.IsExpanded())
		return
	}

	ctv.app.SelectedChannel = c
	ctv.app.SetFocus(ctv.app.MessageInputField)

	title := discord.ChannelToString(c)
	if c.Topic != "" {
		title += " - " + discord.ParseMarkdown(c.Topic)
	}
	ctv.app.MessagesTextView.SetTitle(title)

	go func() {
		ms, err := ctv.app.Session.ChannelMessages(c.ID, ctv.app.Config.General.FetchMessagesLimit, "", "", "")
		if err != nil {
			return
		}

		for i := len(ms) - 1; i >= 0; i-- {
			ctv.app.SelectedChannel.Messages = append(ctv.app.SelectedChannel.Messages, ms[i])
			_, err = ctv.app.MessagesTextView.Write(buildMessage(ctv.app, ms[i]))
			if err != nil {
				return
			}
		}

		ctv.app.MessagesTextView.ScrollToEnd()
	}()
}
