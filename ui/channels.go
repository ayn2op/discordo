package ui

import (
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/rivo/tview"
)

type ChannelsTree struct {
	*tview.TreeView
	app *App
}

func NewChannelsTree(app *App) *ChannelsTree {
	ct := &ChannelsTree{
		TreeView: tview.NewTreeView(),
		app:      app,
	}

	ct.SetRoot(tview.NewTreeNode(""))
	ct.SetTopLevel(1)
	ct.SetSelectedFunc(ct.onSelected)

	ct.SetTitle("Channels")
	ct.SetTitleAlign(tview.AlignLeft)
	ct.SetBorder(true)
	ct.SetBorderPadding(0, 0, 1, 1)

	return ct
}

func (ct *ChannelsTree) onSelected(node *tview.TreeNode) {
	ct.app.SelectedChannel = nil
	ct.app.SelectedMessage = -1
	ct.app.MessagesTextView.
		Highlight().
		Clear().
		SetTitle("")
	ct.app.MessageInputField.SetText("")

	ref := node.GetReference()
	c, err := ct.app.State.Cabinet.Channel(ref.(discord.ChannelID))
	if err != nil {
		return
	}

	// If the channel is a category channel, expend the selected node if it is not expanded already.
	if c.Type == discord.GuildCategory {
		node.SetExpanded(!node.IsExpanded())
		return
	}

	ct.app.SelectedChannel = c
	ct.app.SetFocus(ct.app.MessageInputField)

	title := channelToString(*c)
	if c.Topic != "" {
		title += " - " + parseMarkdown(c.Topic)
	}
	ct.app.MessagesTextView.SetTitle(title)

	go func() {
		// The returned slice will be sorted from latest to oldest.
		ms, err := ct.app.State.Messages(c.ID, ct.app.Config.MessagesLimit)
		if err != nil {
			return
		}

		for i := len(ms) - 1; i >= 0; i-- {
			_, err = ct.app.MessagesTextView.Write(buildMessage(ct.app, ms[i]))
			if err != nil {
				return
			}
		}

		ct.app.MessagesTextView.ScrollToEnd()
	}()
}
