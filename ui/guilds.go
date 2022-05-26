package ui

import (
	"sort"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type GuildsList struct {
	*tview.List
	app *App
}

func NewGuildsList(a *App) *GuildsList {
	l := &GuildsList{
		List: tview.NewList(),
		app:  a,
	}

	l.ShowSecondaryText(false)
	l.SetBorder(true)
	l.SetBorderPadding(0, 0, 1, 1)
	l.SetTitle(" Guilds ")
	l.SetSelectedFunc(l.onSelected)

	l.SetSelectedBackgroundColor(tview.Styles.PrimitiveBackgroundColor)
	l.SetMainTextColor(tcell.GetColor(a.config.Theme.GuildsList.ItemForeground))
	l.SetSelectedTextColor(tcell.GetColor(a.config.Theme.GuildsList.SelectedItemForeground))

	return l
}

func (l *GuildsList) onSelected(index int, _ string, _ string, _ rune) {
	// Cleanup
	l.app.channelsTree.root.ClearChildren()

	cs := l.app.state.Ready().Guilds[index].Channels
	sort.Slice(cs, func(i, j int) bool {
		return cs[i].Position < cs[j].Position
	})

	l.createOrphanChannels(cs)
	l.createCategoryChannels(cs)
	l.createChildrenChannels(cs)
}

func (l *GuildsList) createOrphanChannels(cs []discord.Channel) {
	for _, c := range cs {
		if (c.Type == discord.GuildText || c.Type == discord.GuildNews) && (!c.ParentID.IsValid()) {
			channelNode := tview.NewTreeNode(c.Name).
				SetReference(c.ID)
			l.app.channelsTree.root.AddChild(channelNode)
		}
	}
}

func (l *GuildsList) createCategoryChannels(cs []discord.Channel) {
CATEGORY:
	for _, c := range cs {
		if c.Type == discord.GuildCategory {
			for _, nestedChannel := range cs {
				if nestedChannel.ParentID == c.ID {
					channelTreeNode := tview.NewTreeNode(c.Name).
						SetReference(c.ID)
					l.app.channelsTree.root.AddChild(channelTreeNode)
					continue CATEGORY
				}
			}

			channelNode := tview.NewTreeNode(c.Name).
				SetReference(c.ID)
			l.app.channelsTree.root.AddChild(channelNode)
		}
	}
}

func (l *GuildsList) createChildrenChannels(cs []discord.Channel) {
	for _, c := range cs {
		if (c.Type == discord.GuildText || c.Type == discord.GuildNews) && (c.ParentID.IsValid()) {
			var parentNode *tview.TreeNode
			l.app.channelsTree.root.Walk(func(node, _ *tview.TreeNode) bool {
				if node.GetReference() == c.ParentID {
					parentNode = node
					return false
				}

				return true
			})

			if parentNode != nil {
				channelNode := tview.NewTreeNode(c.Name).
					SetReference(c.ID)
				parentNode.AddChild(channelNode)
			}
		}
	}
}
