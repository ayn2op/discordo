package cmd

import (
	"fmt"
	"log"
	"sort"
	"strings"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type GuildsTree struct {
	*tview.TreeView

	selectedChannelID discord.ChannelID
}

func newGuildsTree() *GuildsTree {
	gt := &GuildsTree{
		TreeView: tview.NewTreeView(),
	}

	root := tview.NewTreeNode("")
	gt.SetRoot(root)

	gt.SetTopLevel(1)
	gt.SetGraphics(cfg.Theme.GuildsTree.Graphics)
	gt.SetBackgroundColor(tcell.GetColor(cfg.Theme.BackgroundColor))
	gt.SetSelectedFunc(gt.onSelected)

	gt.SetTitle("Guilds")
	gt.SetTitleColor(tcell.GetColor(cfg.Theme.TitleColor))
	gt.SetTitleAlign(tview.AlignLeft)

	p := cfg.Theme.BorderPadding
	gt.SetBorder(cfg.Theme.Border)
	gt.SetBorderColor(tcell.GetColor(cfg.Theme.BorderColor))
	gt.SetBorderPadding(p[0], p[1], p[2], p[3])

	gt.SetInputCapture(gt.onInputCapture)
	return gt
}

func (gt *GuildsTree) createFolderNode(folder gateway.GuildFolder) {
	var name string
	if folder.Name == "" {
		name = "Folder"
	} else {
		name = fmt.Sprintf("[%s]%s[-]", folder.Color.String(), folder.Name)
	}

	root := gt.GetRoot()
	folderNode := tview.NewTreeNode(name)
	folderNode.SetExpanded(cfg.Theme.GuildsTree.AutoExpandFolders)
	root.AddChild(folderNode)

	for _, gID := range folder.GuildIDs {
		g, err := discordState.Cabinet.Guild(gID)
		if err != nil {
			log.Printf("guild %v not found in state: %v\n", gID, err)
			continue
		}

		gt.createGuildNode(folderNode, *g)
	}
}

func (gt *GuildsTree) createGuildNode(n *tview.TreeNode, g discord.Guild) {
	guildNode := tview.NewTreeNode(g.Name)
	guildNode.SetReference(g.ID)
	n.AddChild(guildNode)
}

func (gt *GuildsTree) channelToString(c discord.Channel) string {
	var s string
	switch c.Type {
	case discord.GuildText:
		s = "#" + c.Name
	case discord.DirectMessage:
		r := c.DMRecipients[0]
		s = r.Tag()
	case discord.GuildVoice:
		s = "v-" + c.Name
	case discord.GroupDM:
		s = c.Name
		// If the name of the channel is empty, use the recipients' tags
		if s == "" {
			rs := make([]string, len(c.DMRecipients))
			for _, r := range c.DMRecipients {
				rs = append(rs, r.Tag())
			}

			s = strings.Join(rs, ", ")
		}
	case discord.GuildAnnouncement:
		s = "a-" + c.Name
	case discord.GuildStore:
		s = "s-" + c.Name
	case discord.GuildForum:
		s = "f-" + c.Name
	default:
		s = c.Name
	}

	return s
}

func (gt *GuildsTree) createChannelNode(n *tview.TreeNode, c discord.Channel) *tview.TreeNode {
	if c.Type != discord.DirectMessage && c.Type != discord.GroupDM {
		ps, err := discordState.Permissions(c.ID, discordState.Ready().User.ID)
		if err != nil {
			log.Println(err)
			return nil
		}

		if !ps.Has(discord.PermissionViewChannel) {
			return nil
		}
	}

	channelNode := tview.NewTreeNode(gt.channelToString(c))
	channelNode.SetReference(c.ID)
	n.AddChild(channelNode)
	return channelNode
}

func (gt *GuildsTree) createChannelNodes(n *tview.TreeNode, cs []discord.Channel) {
	for _, c := range cs {
		if c.Type != discord.GuildCategory && !c.ParentID.IsValid() {
			gt.createChannelNode(n, c)
		}
	}

PARENT_CHANNELS:
	for _, c := range cs {
		if c.Type == discord.GuildCategory {
			for _, nested := range cs {
				if nested.ParentID == c.ID {
					gt.createChannelNode(n, c)
					continue PARENT_CHANNELS
				}
			}
		}
	}

	for _, c := range cs {
		if c.ParentID.IsValid() {
			var parent *tview.TreeNode
			n.Walk(func(node, _ *tview.TreeNode) bool {
				if node.GetReference() == c.ParentID {
					parent = node
					return false
				}

				return true
			})

			if parent != nil {
				gt.createChannelNode(parent, c)
			}
		}
	}
}

func (gt *GuildsTree) onSelected(n *tview.TreeNode) {
	gt.selectedChannelID = 0

	mainFlex.messagesText.reset()
	mainFlex.messageInput.reset()

	if len(n.GetChildren()) != 0 {
		n.SetExpanded(!n.IsExpanded())
		return
	}

	switch ref := n.GetReference().(type) {
	case discord.GuildID:
		cs, err := discordState.Cabinet.Channels(ref)
		if err != nil {
			log.Println(err)
			return
		}

		sort.Slice(cs, func(i, j int) bool {
			return cs[i].Position < cs[j].Position
		})

		gt.createChannelNodes(n, cs)
	case discord.ChannelID:
		mainFlex.messagesText.drawMsgs(ref)
		mainFlex.messagesText.ScrollToEnd()

		c, err := discordState.Cabinet.Channel(ref)
		if err != nil {
			log.Println(err)
			return
		}

		mainFlex.messagesText.SetTitle(gt.channelToString(*c))

		gt.selectedChannelID = ref
		app.SetFocus(mainFlex.messageInput)
	case nil: // Direct messages
		cs, err := discordState.Cabinet.PrivateChannels()
		if err != nil {
			log.Println(err)
			return
		}

		sort.Slice(cs, func(i, j int) bool {
			return cs[i].LastMessageID > cs[j].LastMessageID
		})

		for _, c := range cs {
			gt.createChannelNode(n, c)
		}
	}
}

func (gt *GuildsTree) onInputCapture(event *tcell.EventKey) *tcell.EventKey {
	switch event.Name() {
	case cfg.Keys.SelectPrevious:
		return tcell.NewEventKey(tcell.KeyUp, 0, tcell.ModNone)
	case cfg.Keys.SelectNext:
		return tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModNone)
	case cfg.Keys.SelectFirst:
		return tcell.NewEventKey(tcell.KeyHome, 0, tcell.ModNone)
	case cfg.Keys.SelectLast:
		return tcell.NewEventKey(tcell.KeyEnd, 0, tcell.ModNone)

	case cfg.Keys.GuildsTree.SelectCurrent:
		return tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone)
	}

	return nil
}
