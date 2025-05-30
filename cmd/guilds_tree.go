package cmd

import (
	"fmt"
	"log/slog"
	"sort"
	"strings"

	"github.com/atotto/clipboard"
	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/discordo/internal/ui"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type GuildsTree struct {
	*tview.TreeView
	cfg               *config.Config
	app               *tview.Application
	selectedChannelID discord.ChannelID
	selectedGuildID   discord.GuildID
}

func newGuildsTree(app *tview.Application, cfg *config.Config) *GuildsTree {
	gt := &GuildsTree{
		TreeView: tview.NewTreeView(),
		cfg:      cfg,
		app:      app,
	}

	gt.Box = ui.NewConfiguredBox(gt.Box, &cfg.Theme)

	gt.
		SetRoot(tview.NewTreeNode("")).
		SetTopLevel(1).
		SetGraphics(cfg.Theme.GuildsTree.Graphics).
		SetSelectedFunc(gt.onSelected).
		SetTitle("Guilds").
		SetInputCapture(gt.onInputCapture)

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
	folderNode.SetExpanded(gt.cfg.Theme.GuildsTree.AutoExpandFolders)
	root.AddChild(folderNode)

	for _, gID := range folder.GuildIDs {
		g, err := discordState.Cabinet.Guild(gID)
		if err != nil {
			slog.Info("failed to get guild from state", "guild_id", gID, "err", err)
			continue
		}

		gt.createGuildNode(folderNode, *g)
	}
}

func (gt *GuildsTree) createGuildNode(n *tview.TreeNode, g discord.Guild) {
	guildNode := tview.NewTreeNode(g.Name)
	guildNode.SetReference(g.ID)
	guildNode.SetColor(tcell.GetColor(gt.cfg.Theme.GuildsTree.GuildColor))
	n.AddChild(guildNode)
}

func (gt *GuildsTree) channelToString(c discord.Channel) string {
	switch c.Type {
	case discord.DirectMessage, discord.GroupDM:
		if c.Name != "" {
			return c.Name
		}

		recipients := make([]string, len(c.DMRecipients))
		for i, r := range c.DMRecipients {
			recipients[i] = r.DisplayOrUsername()
		}

		return strings.Join(recipients, ", ")

	case discord.GuildText:
		return "#" + c.Name
	case discord.GuildVoice, discord.GuildStageVoice:
		return "v-" + c.Name
	case discord.GuildAnnouncement:
		return "a-" + c.Name
	case discord.GuildStore:
		return "s-" + c.Name
	case discord.GuildForum:
		return "f-" + c.Name
	default:
		return c.Name
	}
}

func (gt *GuildsTree) createChannelNode(n *tview.TreeNode, c discord.Channel) *tview.TreeNode {
	if c.Type != discord.DirectMessage && c.Type != discord.GroupDM {
		ps, err := discordState.Permissions(c.ID, discordState.Ready().User.ID)
		if err != nil {
			slog.Error("failed to get permissions", "err", err, "channel_id", c.ID)
			return nil
		}

		if !ps.Has(discord.PermissionViewChannel) {
			return nil
		}
	}

	channelNode := tview.NewTreeNode(gt.channelToString(c))
	channelNode.SetReference(c.ID)
	channelNode.SetColor(tcell.GetColor(gt.cfg.Theme.GuildsTree.ChannelColor))
	n.AddChild(channelNode)
	return channelNode
}

func (gt *GuildsTree) createChannelNodes(n *tview.TreeNode, cs []discord.Channel) {
	var orphanChs []discord.Channel
	for _, ch := range cs {
		if ch.Type != discord.GuildCategory && !ch.ParentID.IsValid() {
			orphanChs = append(orphanChs, ch)
		}
	}

	for _, c := range orphanChs {
		gt.createChannelNode(n, c)
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

	app.messagesText.reset()
	app.messageInput.reset()

	if len(n.GetChildren()) != 0 {
		n.SetExpanded(!n.IsExpanded())
		return
	}

	switch ref := n.GetReference().(type) {
	case discord.GuildID:
		go discordState.MemberState.Subscribe(ref)

		cs, err := discordState.Cabinet.Channels(ref)
		if err != nil {
			slog.Error("failed to get channels", "err", err, "guild_id", ref)
			return
		}

		sort.Slice(cs, func(i, j int) bool {
			return cs[i].Position < cs[j].Position
		})

		gt.createChannelNodes(n, cs)
	case discord.ChannelID:
		c, err := discordState.Cabinet.Channel(ref)
		if err != nil {
			slog.Error("failed to get channel", "channel_id", ref)
			return
		}

		app.messagesText.drawMsgs(c.ID)
		app.messagesText.ScrollToEnd()
		app.messagesText.SetTitle(gt.channelToString(*c))

		gt.selectedChannelID = c.ID
		gt.selectedGuildID = c.GuildID
		gt.app.SetFocus(app.messageInput)
	case nil: // Direct messages
		cs, err := discordState.PrivateChannels()
		if err != nil {
			slog.Error("failed to get private channels", "err", err)
			return
		}

		sort.Slice(cs, func(a, b int) bool {
			msgID := func(ch discord.Channel) discord.MessageID {
				if ch.LastMessageID.IsValid() {
					return ch.LastMessageID
				}
				return discord.MessageID(ch.ID)
			}
			return msgID(cs[a]) > msgID(cs[b])
		})

		for _, c := range cs {
			gt.createChannelNode(n, c)
		}
	}
}

func (gt *GuildsTree) collapseParentNode(node *tview.TreeNode) {
	gt.
		GetRoot().
		Walk(func(n, parent *tview.TreeNode) bool {
			if n == node && parent.GetLevel() != 0 {
				parent.Collapse()
				gt.SetCurrentNode(parent)
				return false
			}

			return true
		})
}

func (gt *GuildsTree) onInputCapture(event *tcell.EventKey) *tcell.EventKey {
	switch event.Name() {
	case gt.cfg.Keys.GuildsTree.CollapseParentNode:
		gt.collapseParentNode(gt.GetCurrentNode())
		return nil
	case gt.cfg.Keys.GuildsTree.MoveToParentNode:
		return tcell.NewEventKey(tcell.KeyRune, 'K', tcell.ModNone)

	case gt.cfg.Keys.GuildsTree.SelectPrevious:
		return tcell.NewEventKey(tcell.KeyUp, 0, tcell.ModNone)
	case gt.cfg.Keys.GuildsTree.SelectNext:
		return tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModNone)
	case gt.cfg.Keys.GuildsTree.SelectFirst:
		gt.Move(gt.GetRowCount() * -1)
		// return tcell.NewEventKey(tcell.KeyHome, 0, tcell.ModNone)
	case gt.cfg.Keys.GuildsTree.SelectLast:
		gt.Move(gt.GetRowCount())
		// return tcell.NewEventKey(tcell.KeyEnd, 0, tcell.ModNone)

	case gt.cfg.Keys.GuildsTree.SelectCurrent:
		return tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone)

	case gt.cfg.Keys.GuildsTree.YankID:
		node := gt.GetCurrentNode()
		if node == nil {
			return nil
		}

		// Reference of a tree node in the guilds tree is its ID.
		// discord.Snowflake (discord.GuildID and discord.ChannelID) have the String method.
		if id, ok := node.GetReference().(fmt.Stringer); ok {
			go clipboard.WriteAll(id.String())
		}
	}

	return nil
}
