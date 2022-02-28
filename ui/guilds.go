package ui

import (
	"sort"

	"github.com/ayntgl/discordgo"
	"github.com/ayntgl/discordo/discord"
	"github.com/rivo/tview"
)

type GuildsList struct {
	*tview.List
	app *App
}

func NewGuildsList(app *App) *GuildsList {
	gl := &GuildsList{
		List: tview.NewList(),
		app:  app,
	}

	gl.ShowSecondaryText(false)
	gl.SetTitle("Guilds")
	gl.SetTitleAlign(tview.AlignLeft)
	gl.SetBorder(true)
	gl.SetBorderPadding(0, 0, 1, 1)
	gl.SetSelectedFunc(gl.onSelected)
	return gl
}

func (gl *GuildsList) onSelected(idx int, mainText string, secondaryText string, shortcut rune) {
	rootTreeNode := gl.app.ChannelsTreeView.GetRoot()
	rootTreeNode.ClearChildren()
	gl.app.SelectedMessage = -1
	gl.app.MessagesTextView.
		Highlight().
		Clear()
	gl.app.MessageInputField.SetText("")

	if mainText == "Direct Messages" {
		cs := gl.app.Session.State.PrivateChannels
		sort.Slice(cs, func(i, j int) bool {
			return cs[i].LastMessageID > cs[j].LastMessageID
		})

		for _, c := range cs {
			channelTreeNode := tview.NewTreeNode(discord.ChannelToString(c)).
				SetReference(c.ID)
			rootTreeNode.AddChild(channelTreeNode)
		}
	} else { // Guild
		cs := gl.app.Session.State.Guilds[idx].Channels
		sort.Slice(cs, func(i, j int) bool {
			return cs[i].Position < cs[j].Position
		})

		for _, c := range cs {
			if (c.Type == discordgo.ChannelTypeGuildText || c.Type == discordgo.ChannelTypeGuildNews) && (c.ParentID == "") {
				channelTreeNode := tview.NewTreeNode(discord.ChannelToString(c)).
					SetReference(c.ID)
				rootTreeNode.AddChild(channelTreeNode)
			}
		}

	CATEGORY:
		for _, c := range cs {
			if c.Type == discordgo.ChannelTypeGuildCategory {
				for _, nestedChannel := range cs {
					if nestedChannel.ParentID == c.ID {
						channelTreeNode := tview.NewTreeNode(c.Name).
							SetReference(c.ID)
						rootTreeNode.AddChild(channelTreeNode)
						continue CATEGORY
					}
				}

				channelTreeNode := tview.NewTreeNode(c.Name).
					SetReference(c.ID)
				rootTreeNode.AddChild(channelTreeNode)
			}
		}

		for _, c := range cs {
			if (c.Type == discordgo.ChannelTypeGuildText || c.Type == discordgo.ChannelTypeGuildNews) && (c.ParentID != "") {
				var parentTreeNode *tview.TreeNode
				rootTreeNode.Walk(func(node, _ *tview.TreeNode) bool {
					if node.GetReference() == c.ParentID {
						parentTreeNode = node
						return false
					}

					return true
				})

				if parentTreeNode != nil {
					channelTreeNode := tview.NewTreeNode(discord.ChannelToString(c)).
						SetReference(c.ID)
					parentTreeNode.AddChild(channelTreeNode)
				}
			}
		}
	}

	gl.app.ChannelsTreeView.SetCurrentNode(rootTreeNode)
	gl.app.SetFocus(gl.app.ChannelsTreeView)
}
