package cmd

import (
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/discordo/internal/ui"
	"github.com/ayn2op/tview"
	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/diamondburned/ningen/v3"
	"github.com/gdamore/tcell/v2"
)

type guildsTree struct {
	*tview.TreeView
	cfg               *config.Config
	selectedChannelID discord.ChannelID
	selectedGuildID   discord.GuildID
}

func newGuildsTree(cfg *config.Config) *guildsTree {
	gt := &guildsTree{
		TreeView: tview.NewTreeView(),
		cfg:      cfg,
	}

	gt.Box = ui.ConfigureBox(gt.Box, &cfg.Theme)
	gt.
		SetRoot(tview.NewTreeNode("")).
		SetTopLevel(1).
		SetGraphics(cfg.Theme.GuildsTree.Graphics).
		SetGraphicsColor(tcell.GetColor(cfg.Theme.GuildsTree.GraphicsColor)).
		SetSelectedFunc(gt.onSelected).
		SetTitle("Guilds").
		SetInputCapture(gt.onInputCapture)

	return gt
}

func (gt *guildsTree) createFolderNode(folder gateway.GuildFolder) {
	name := "Folder"
	if folder.Name != "" {
		name = fmt.Sprintf("[%s]%s[-]", folder.Color.String(), folder.Name)
	}

	folderNode := tview.NewTreeNode(name).SetExpanded(gt.cfg.Theme.GuildsTree.AutoExpandFolders)
	gt.GetRoot().AddChild(folderNode)

	for _, gID := range folder.GuildIDs {
		guild, err := discordState.Cabinet.Guild(gID)
		if err != nil {
			slog.Error("failed to get guild from state", "guild_id", gID, "err", err)
			continue
		}

		gt.createGuildNode(folderNode, *guild)
	}
}

func (gt *guildsTree) createGuildNode(n *tview.TreeNode, g discord.Guild) {
	style := gt.cfg.Theme.GuildsTree.GuildStyle.Style
	guildNode := tview.NewTreeNode(g.Name).
		SetReference(g.ID).
		SetTextStyle(style).
		SetSelectedTextStyle(style.Reverse(true))
	n.AddChild(guildNode)
}




func (gt *guildsTree) refreshChannelDisplay(channelID discord.ChannelID) {
	gt.GetRoot().Walk(func(node, _ *tview.TreeNode) bool {
		if node.GetReference() == channelID {
			channel, err := discordState.Cabinet.Channel(channelID)
			if err != nil {
				return true
			}
			node.SetText(gt.getChannelDisplayName(*channel))
			return false
		}
		return true
	})
}

func (gt *guildsTree) channelToString(channel discord.Channel) string {
	var name string
	switch channel.Type {
	case discord.DirectMessage, discord.GroupDM:
		if channel.Name != "" {
			name = channel.Name
		} else {
      recipients := make([]string, len(channel.DMRecipients))
		  for i, r := range channel.DMRecipients {
			  recipients[i] = ui.PreferredName(r, gt.cfg.Theme)
      }
      
			name = strings.Join(recipients, ", ")
    }
	case discord.GuildText:
		name = "#" + channel.Name
	case discord.GuildVoice, discord.GuildStageVoice:
		name = "v-" + channel.Name
	case discord.GuildAnnouncement:
		name = "a-" + channel.Name
	case discord.GuildStore:
		name = "s-" + channel.Name
	case discord.GuildForum:
		name = "f-" + channel.Name
	default:
		name = channel.Name
	}

	return name
}

func (gt *guildsTree) getChannelDisplayName(channel discord.Channel) string {
	name := gt.channelToString(channel)
	
	opts := ningen.UnreadOpts{
		IncludeMutedCategories: true,
	}
	indication := discordState.ChannelIsUnread(channel.ID, opts)
	switch indication {
	case ningen.ChannelMentioned:
		return name + " @"
	case ningen.ChannelUnread:
		return name + " ●"
	}
	
	return name
}

func (gt *guildsTree) markChannelAsRead(channelID discord.ChannelID) {
	lastMsgID := discordState.LastMessage(channelID)
	if !lastMsgID.IsValid() {
		slog.Debug("no messages to mark as read", "channel_id", channelID)
		return
	}
	
	gt.ackChannel(channelID, lastMsgID)
}

func (gt *guildsTree) ackChannel(channelID discord.ChannelID, messageID discord.MessageID) {
	
	ack := &api.Ack{}
        err := discordState.Ack(channelID, messageID, ack)
        if err != nil {
                slog.Debug("failed to acknowledge channel", "channel_id", channelID, "message_id", messageID, "err", err)
                return
        }
	
	slog.Debug("marked channel as read", "channel_id", channelID, "message_id", messageID)
}


func (gt *guildsTree) findUnreadChannelsAcrossAllGuilds() []discord.ChannelID {
	var mentionedChannels []discord.ChannelID
	var unreadChannels []discord.ChannelID
	opts := ningen.UnreadOpts{
		IncludeMutedCategories: true,
	}
	
	gt.GetRoot().Walk(func(node, _ *tview.TreeNode) bool {
		if guildID, ok := node.GetReference().(discord.GuildID); ok {
			channels, err := discordState.Cabinet.Channels(guildID)
			if err != nil {
				return true
			}
			
			sort.Slice(channels, func(i, j int) bool {
				return channels[i].Position < channels[j].Position
			})
			
			for _, channel := range channels {
				indication := discordState.ChannelIsUnread(channel.ID, opts)
				if indication == ningen.ChannelMentioned {
					mentionedChannels = append(mentionedChannels, channel.ID)
				} else if indication == ningen.ChannelUnread {
					unreadChannels = append(unreadChannels, channel.ID)
				}
			}
		}
		return true
	})
	
	privateChannels, err := discordState.Cabinet.PrivateChannels()
	if err == nil {
		for _, channel := range privateChannels {
			indication := discordState.ChannelIsUnread(channel.ID, opts)
			if indication == ningen.ChannelMentioned {
				mentionedChannels = append(mentionedChannels, channel.ID)
			} else if indication == ningen.ChannelUnread {
				unreadChannels = append(unreadChannels, channel.ID)
			}
		}
	}
	
	return append(mentionedChannels, unreadChannels...)
}


func (gt *guildsTree) expandAllGuildsWithUnread() {
	unreadChannels := gt.findUnreadChannelsAcrossAllGuilds()
	
	guildIDsWithUnread := make(map[discord.GuildID]bool)
	for _, channelID := range unreadChannels {
		channel, err := discordState.Cabinet.Channel(channelID)
		if err != nil {
			continue
		}
		if channel.GuildID.IsValid() {
			guildIDsWithUnread[channel.GuildID] = true
		}
	}
	
	gt.GetRoot().Walk(func(node, _ *tview.TreeNode) bool {
		if guildID, ok := node.GetReference().(discord.GuildID); ok {
			if guildIDsWithUnread[guildID] {
				if !node.IsExpanded() {
					node.SetExpanded(true)
				}
				if len(node.GetChildren()) == 0 {
					gt.onSelected(node)
				}
			}
		}
		return true
	})
	
	hasUnreadDMs := false
	for _, channelID := range unreadChannels {
		channel, err := discordState.Cabinet.Channel(channelID)
		if err != nil {
			continue
		}
		if !channel.GuildID.IsValid() {
			hasUnreadDMs = true
			break
		}
	}
	
	if hasUnreadDMs {
		gt.GetRoot().Walk(func(node, _ *tview.TreeNode) bool {
			if node.GetReference() == nil && strings.Contains(strings.ToLower(node.GetText()), "direct") {
				if !node.IsExpanded() {
					node.SetExpanded(true)
				}
				// Load channels if not already loaded
				if len(node.GetChildren()) == 0 {
					gt.onSelected(node)
				}
				return false
			}
			return true
		})
	}
}

func (gt *guildsTree) findDirectionalUnread(unreadChannels []discord.ChannelID, currentChannelID discord.ChannelID, direction int) discord.ChannelID {
	if !currentChannelID.IsValid() {
		return discord.ChannelID(0)
	}
	
	unreadSet := make(map[discord.ChannelID]bool)
	for _, id := range unreadChannels {
		unreadSet[id] = true
	}
	
	var allChannelsInOrder []discord.ChannelID
	gt.GetRoot().Walk(func(node, _ *tview.TreeNode) bool {
		if channelID, ok := node.GetReference().(discord.ChannelID); ok {
			allChannelsInOrder = append(allChannelsInOrder, channelID)
		}
		return true
	})
	
	currentIndex := -1
	for i, channelID := range allChannelsInOrder {
		if channelID == currentChannelID {
			currentIndex = i
			break
		}
	}
	
	if currentIndex == -1 {
		return discord.ChannelID(0)
	}
	
	if direction > 0 {
		for i := currentIndex + 1; i < len(allChannelsInOrder); i++ {
			if unreadSet[allChannelsInOrder[i]] {
				return allChannelsInOrder[i]
			}
		}
	} else {
		for i := currentIndex - 1; i >= 0; i-- {
			if unreadSet[allChannelsInOrder[i]] {
				return allChannelsInOrder[i]
			}
		}
	}
	
	return discord.ChannelID(0)
}

func (gt *guildsTree) jumpToUnreadChannel(direction int) {
	gt.expandAllGuildsWithUnread()
	
	unreadChannels := gt.findUnreadChannelsAcrossAllGuilds()
	
	if len(unreadChannels) == 0 {
		gt.SetTitle("Guilds (no unread messages)")
		go func() {
			time.Sleep(1500 * time.Millisecond)
			gt.SetTitle("Guilds")
		}()
		return 
	}

	currentNode := gt.GetCurrentNode()
	var currentChannelID discord.ChannelID
	if currentNode != nil {
		if channelID, ok := currentNode.GetReference().(discord.ChannelID); ok {
			currentChannelID = channelID
		}
	}
	
	if !currentChannelID.IsValid() && gt.selectedChannelID.IsValid() {
		currentChannelID = gt.selectedChannelID
	}

	currentIndex := -1
	for i, channelID := range unreadChannels {
		if channelID == currentChannelID {
			currentIndex = i
			break
		}
	}

	var targetChannelID discord.ChannelID
	if currentIndex == -1 {
		targetChannelID = gt.findDirectionalUnread(unreadChannels, currentChannelID, direction)
		
		if !targetChannelID.IsValid() {
			if direction > 0 {
				targetChannelID = unreadChannels[0]
			} else {
				targetChannelID = unreadChannels[len(unreadChannels)-1]
			}
		}
	} else {
		var targetIndex int
		if direction > 0 {
			targetIndex = (currentIndex + 1) % len(unreadChannels)
		} else {
			targetIndex = (currentIndex - 1 + len(unreadChannels)) % len(unreadChannels)
		}
		targetChannelID = unreadChannels[targetIndex]
	}

	gt.GetRoot().Walk(func(node, _ *tview.TreeNode) bool {
		if channelID, ok := node.GetReference().(discord.ChannelID); ok && channelID == targetChannelID {
			gt.SetCurrentNode(node)
			return false
		}
		return true
	})
}


func (gt *guildsTree) createChannelNode(node *tview.TreeNode, channel discord.Channel) {
	if channel.Type != discord.DirectMessage && channel.Type != discord.GroupDM {
		perms, err := discordState.Permissions(channel.ID, discordState.Ready().User.ID)
		if err != nil {
			slog.Error("failed to get permissions", "err", err, "channel_id", channel.ID)
			return
		}

		if !perms.Has(discord.PermissionViewChannel) {
			return
		}
	}


	style := gt.cfg.Theme.GuildsTree.ChannelStyle.Style
	displayName := gt.getChannelDisplayName(channel)
	
	channelNode := tview.NewTreeNode(displayName).
		SetReference(channel.ID).
		SetTextStyle(style).
		SetSelectedTextStyle(style.Reverse(true))
	node.AddChild(channelNode)
}

func (gt *guildsTree) createChannelNodes(node *tview.TreeNode, channels []discord.Channel) {
	var orphanChs []discord.Channel
	for _, ch := range channels {
		if ch.Type != discord.GuildCategory && !ch.ParentID.IsValid() {
			orphanChs = append(orphanChs, ch)
		}
	}

	for _, c := range orphanChs {
		gt.createChannelNode(node, c)
	}

PARENT_CHANNELS:
	for _, c := range channels {
		if c.Type == discord.GuildCategory {
			for _, nested := range channels {
				if nested.ParentID == c.ID {
					gt.createChannelNode(node, c)
					continue PARENT_CHANNELS
				}
			}
		}
	}

	for _, channel := range channels {
		if channel.ParentID.IsValid() {
			var parent *tview.TreeNode
			node.Walk(func(node, _ *tview.TreeNode) bool {
				if node.GetReference() == channel.ParentID {
					parent = node
					return false
				}

				return true
			})

			if parent != nil {
				gt.createChannelNode(parent, channel)
			}
		}
	}
}

func (gt *guildsTree) onSelected(node *tview.TreeNode) {
	app.messageInput.reset()

	if len(node.GetChildren()) != 0 {
		node.SetExpanded(!node.IsExpanded())
		return
	}

	switch ref := node.GetReference().(type) {
	case discord.GuildID:
		go discordState.MemberState.Subscribe(ref)

		channels, err := discordState.Cabinet.Channels(ref)
		if err != nil {
			slog.Error("failed to get channels", "err", err, "guild_id", ref)
			return
		}

		sort.Slice(channels, func(i, j int) bool {
			return channels[i].Position < channels[j].Position
		})

		gt.createChannelNodes(node, channels)
	case discord.ChannelID:
		channel, err := discordState.Cabinet.Channel(ref)
		if err != nil {
			slog.Error("failed to get channel", "channel_id", ref)
			return
		}

		gt.markChannelAsRead(channel.ID)
		
		gt.refreshChannelDisplay(channel.ID)

		app.messagesList.reset()
		app.messagesList.drawMsgs(channel.ID)
		app.messagesList.
			ScrollToEnd().
			SetTitle(gt.channelToString(*channel))

		gt.selectedChannelID = channel.ID
		gt.selectedGuildID = channel.GuildID
		app.SetFocus(app.messageInput)
	case nil: // Direct messages
		channels, err := discordState.PrivateChannels()
		if err != nil {
			slog.Error("failed to get private channels", "err", err)
			return
		}

		sort.Slice(channels, func(a, b int) bool {
			msgID := func(ch discord.Channel) discord.MessageID {
				if ch.LastMessageID.IsValid() {
					return ch.LastMessageID
				}
				return discord.MessageID(ch.ID)
			}
			return msgID(channels[a]) > msgID(channels[b])
		})

		for _, c := range channels {
			gt.createChannelNode(node, c)
		}
	}
}

func (gt *guildsTree) collapseParentNode(node *tview.TreeNode) {
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

func (gt *guildsTree) onInputCapture(event *tcell.EventKey) *tcell.EventKey {
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
		gt.yankID()

	case gt.cfg.Keys.GuildsTree.NextUnread:
		gt.jumpToUnreadChannel(1) // direction = 1 for next
		return nil
	case gt.cfg.Keys.GuildsTree.PreviousUnread:
		gt.jumpToUnreadChannel(-1) // direction = -1 for previous
		return nil
	}

	return nil
}

func (gt *guildsTree) yankID() {
	node := gt.GetCurrentNode()
	if node == nil {
		return
	}

	// Reference of a tree node in the guilds tree is its ID.
	// discord.Snowflake (discord.GuildID and discord.ChannelID) have the String method.
	if id, ok := node.GetReference().(fmt.Stringer); ok {
		go func() {
			if err := clipboard.WriteAll(id.String()); err != nil {
				slog.Error("failed to yank ID from guilds tree to clipboard", "err", err)
			}
		}()
	}
}

