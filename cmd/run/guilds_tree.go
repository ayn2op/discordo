package run

import (
	"fmt"
	"log"
	"sort"
	"strings"

	"github.com/ayn2op/discordo/config"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type GuildsTree struct {
	*tview.TreeView

	root              *tview.TreeNode
	selectedChannelID discord.ChannelID
}

func newGuildsTree() *GuildsTree {
	gt := &GuildsTree{
		TreeView: tview.NewTreeView(),

		root: tview.NewTreeNode(""),
	}

	gt.SetTopLevel(1)
	gt.SetRoot(gt.root)
	gt.SetGraphics(config.Current.Theme.GuildsTree.Graphics)
	gt.SetBackgroundColor(tcell.GetColor(config.Current.Theme.BackgroundColor))
	gt.SetSelectedFunc(gt.onSelected)
	gt.SetInputCapture(gt.onInputCapture)

	gt.SetTitle("Guilds")
	gt.SetTitleColor(tcell.GetColor(config.Current.Theme.TitleColor))
	gt.SetTitleAlign(tview.AlignLeft)

	p := config.Current.Theme.BorderPadding
	gt.SetBorder(config.Current.Theme.Border)
	gt.SetBorderColor(tcell.GetColor(config.Current.Theme.BorderColor))
	gt.SetBorderPadding(p[0], p[1], p[2], p[3])

	return gt
}

func (gt *GuildsTree) createGuildFolderNode(parent *tview.TreeNode, gf gateway.GuildFolder) {
	var name string
	if gf.Name != "" {
		name = fmt.Sprintf("[%s]%s[-]", gf.Color.String(), gf.Name)
	} else {
		name = "Folder"
	}

	n := tview.NewTreeNode(name)
	parent.AddChild(n)

	for _, gid := range gf.GuildIDs {
		g, err := discordState.Cabinet.Guild(gid)
		if err != nil {
			log.Println(err)
			continue
		}

		gt.createGuildNode(n, *g)
	}
}

func (gt *GuildsTree) createGuildNode(n *tview.TreeNode, g discord.Guild) {
	gn := tview.NewTreeNode(g.Name)
	gn.SetReference(g.ID)
	n.AddChild(gn)
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

	cn := tview.NewTreeNode(gt.channelToString(c))
	cn.SetReference(c.ID)
	n.AddChild(cn)
	return cn
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
		ms, err := discordState.Messages(ref, config.Current.MessagesLimit)
		if err != nil {
			log.Println(err)
			return
		}

		for i := len(ms) - 1; i >= 0; i-- {
			mainFlex.messagesText.createMessage(ms[i])
		}

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

func (gt *GuildsTree) gotoNode(idx int) *tview.TreeNode {
	child := gt.GetRoot().GetChildren()[idx]
	gt.SetCurrentNode(child)
	return child
}

func (gt *GuildsTree) selectNode(idx int) {
	gt.onSelected(gt.gotoNode(idx))
}

func (gt *GuildsTree) onInputCapture(event *tcell.EventKey) *tcell.EventKey {
	bookmarksLen := len(config.Current.Bookmarks)

	if event.Name() == config.Current.Keys.Bookmark.PassBookmarks{
		gt.gotoNode(bookmarksLen)
		return nil
	} else if bookmarksLen == 0 {
		return event
	} else if event.Name() == config.Current.Keys.Bookmark.FirstBookmark {
		gt.gotoNode(0)
		return nil
	} else {
		for n, keyStroke := range config.Current.Keys.Bookmark.Slots {
			if keyStroke == event.Name() {
				if n < bookmarksLen {
					gt.selectNode(n)
				} else {
					gt.selectNode(bookmarksLen - 1)
				}
				return nil
			}	
		}
	}

	return event
}
