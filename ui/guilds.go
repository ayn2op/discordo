package ui

import (
	"sort"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/rivo/tview"
)

type GuildsList struct {
	*tview.List
	*Core
}

func NewGuildsList(c *Core) *GuildsList {
	gl := &GuildsList{
		List: tview.NewList(),
		Core: c,
	}

	gl.ShowSecondaryText(false)
	gl.SetSelectedFunc(gl.onSelect)

	gl.SetTitle("Guilds")
	gl.SetTitleAlign(tview.AlignLeft)
	gl.SetBorder(true)
	gl.SetBorderPadding(0, 0, 1, 1)

	return gl
}

func (gl *GuildsList) onSelect(index int, mainText string, _ string, _ rune) {
	rootNode := gl.channelsTree.GetRoot()

	cs := gl.state.Ready().Guilds[index].Channels
	sort.Slice(cs, func(i, j int) bool {
		return cs[i].Position < cs[j].Position
	})

	for _, c := range cs {
		if (c.Type == discord.GuildText || c.Type == discord.GuildNews) && (!c.ParentID.IsValid()) {
			channelNode := tview.NewTreeNode(c.Name)
			channelNode.SetReference(c.ID)

			rootNode.AddChild(channelNode)
		}
	}

	gl.channelsTree.SetCurrentNode(rootNode)
}
