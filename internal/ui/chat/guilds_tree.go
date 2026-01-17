package chat

import (
	"cmp"
	"fmt"
	"log/slog"
	"slices"

	"github.com/ayn2op/discordo/internal/clipboard"
	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/discordo/internal/ui"
	"github.com/ayn2op/tview"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/diamondburned/ningen/v3"
	"github.com/gdamore/tcell/v3"
)

type guildsTree struct {
	*tview.TreeView
	cfg      *config.Config
	chatView *View
}

func newGuildsTree(cfg *config.Config, chatView *View) *guildsTree {
	gt := &guildsTree{
		TreeView: tview.NewTreeView(),
		cfg:      cfg,
		chatView: chatView,
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
		name = fmt.Sprintf("[%s]%s[-]", folder.Color, folder.Name)
	}

	folderNode := tview.NewTreeNode(name).SetExpanded(gt.cfg.Theme.GuildsTree.AutoExpandFolders)
	gt.GetRoot().AddChild(folderNode)

	for _, gID := range folder.GuildIDs {
		guild, err := gt.chatView.state.Cabinet.Guild(gID)
		if err != nil {
			slog.Error("failed to get guild from state", "guild_id", gID, "err", err)
			continue
		}

		gt.createGuildNode(folderNode, *guild)
	}
}

func (gt *guildsTree) unreadStyle(indication ningen.UnreadIndication) tcell.Style {
	var style tcell.Style
	switch indication {
	case ningen.ChannelRead:
		style = style.Dim(true)
	case ningen.ChannelMentioned:
		style = style.Underline(true)
		fallthrough
	case ningen.ChannelUnread:
		style = style.Bold(true)
	}

	return style
}

func (gt *guildsTree) getGuildNodeStyle(guildID discord.GuildID) tcell.Style {
	indication := gt.chatView.state.GuildIsUnread(guildID, ningen.GuildUnreadOpts{UnreadOpts: ningen.UnreadOpts{IncludeMutedCategories: true}})
	return gt.unreadStyle(indication)
}

func (gt *guildsTree) getChannelNodeStyle(channelID discord.ChannelID) tcell.Style {
	indication := gt.chatView.state.ChannelIsUnread(channelID, ningen.UnreadOpts{IncludeMutedCategories: true})
	return gt.unreadStyle(indication)
}

func (gt *guildsTree) createGuildNode(n *tview.TreeNode, guild discord.Guild) {
	guildNode := tview.NewTreeNode(guild.Name).
		SetReference(guild.ID).
		SetTextStyle(gt.getGuildNodeStyle(guild.ID))
	n.AddChild(guildNode)
}

func (gt *guildsTree) createChannelNode(node *tview.TreeNode, channel discord.Channel) {
	if channel.Type != discord.DirectMessage && channel.Type != discord.GroupDM && !gt.chatView.state.HasPermissions(channel.ID, discord.PermissionViewChannel) {
		return
	}

	channelNode := tview.NewTreeNode(ui.ChannelToString(channel)).
		SetReference(channel.ID).
		SetTextStyle(gt.getChannelNodeStyle(channel.ID))
	node.AddChild(channelNode)
}

func (gt *guildsTree) createChannelNodes(node *tview.TreeNode, channels []discord.Channel) {
	for _, channel := range channels {
		if channel.Type != discord.GuildCategory && !channel.ParentID.IsValid() {
			gt.createChannelNode(node, channel)
		}
	}

PARENT_CHANNELS:
	for _, channel := range channels {
		if channel.Type == discord.GuildCategory {
			for _, nested := range channels {
				if nested.ParentID == channel.ID {
					gt.createChannelNode(node, channel)
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
	if len(node.GetChildren()) != 0 {
		node.SetExpanded(!node.IsExpanded())
		return
	}

	switch ref := node.GetReference().(type) {
	case discord.GuildID:
		go gt.chatView.state.MemberState.Subscribe(ref)

		channels, err := gt.chatView.state.Cabinet.Channels(ref)
		if err != nil {
			slog.Error("failed to get channels", "err", err, "guild_id", ref)
			return
		}

		slices.SortFunc(channels, func(a, b discord.Channel) int {
			return cmp.Compare(a.Position, b.Position)
		})

		gt.createChannelNodes(node, channels)
	case discord.ChannelID:
		channel, err := gt.chatView.state.Cabinet.Channel(ref)
		if err != nil {
			slog.Error("failed to get channel", "channel_id", ref)
			return
		}

		// Handle forum channels differently - they contain threads, not direct messages
		if channel.Type == discord.GuildForum {
			// Get all channels from the guild - this includes active threads from GuildCreateEvent
			allChannels, err := gt.chatView.state.Cabinet.Channels(channel.GuildID)
			if err != nil {
				slog.Error("failed to get channels for forum threads", "err", err, "guild_id", channel.GuildID)
				return
			}

			// Filter for threads that belong to this forum channel
			var forumThreads []discord.Channel
			for _, ch := range allChannels {
				if ch.ParentID == channel.ID && (ch.Type == discord.GuildPublicThread ||
					ch.Type == discord.GuildPrivateThread ||
					ch.Type == discord.GuildAnnouncementThread) {
					forumThreads = append(forumThreads, ch)
				}
			}

			// Add threads as child nodes
			for _, thread := range forumThreads {
				gt.createChannelNode(node, thread)
			}

			// Expand the node to show threads
			node.SetExpanded(true)
			return
		}

		go gt.chatView.state.ReadState.MarkRead(channel.ID, channel.LastMessageID)

		limit := gt.cfg.MessagesLimit
		messages, err := gt.chatView.state.Messages(channel.ID, uint(limit))
		if err != nil {
			slog.Error("failed to get messages", "err", err, "channel_id", channel.ID, "limit", limit)
			return
		}

		if guildID := channel.GuildID; guildID.IsValid() {
			gt.chatView.messagesList.requestGuildMembers(guildID, messages)
		}

		gt.chatView.SetSelectedChannel(channel)
		gt.chatView.clearTypers()
		gt.chatView.messageInput.stopTypingTimer()

		gt.chatView.messagesList.reset()
		gt.chatView.messagesList.setTitle(*channel)
		gt.chatView.messagesList.setMessages(messages)
		gt.chatView.messagesList.ScrollToEnd()

		hasNoPerm := channel.Type != discord.DirectMessage && channel.Type != discord.GroupDM && !gt.chatView.state.HasPermissions(channel.ID, discord.PermissionSendMessages)
		gt.chatView.messageInput.SetDisabled(hasNoPerm)
		if hasNoPerm {
			gt.chatView.messageInput.SetPlaceholder("You do not have permission to send messages in this channel.")
		} else {
			gt.chatView.messageInput.SetPlaceholder("Message...")
			if gt.cfg.AutoFocus {
				gt.chatView.app.SetFocus(gt.chatView.messageInput)
			}
		}

	case nil: // Direct messages folder
		channels, err := gt.chatView.state.PrivateChannels()
		if err != nil {
			slog.Error("failed to get private channels", "err", err)
			return
		}

		msgID := func(ch discord.Channel) discord.MessageID {
			if ch.LastMessageID.IsValid() {
				return ch.LastMessageID
			}
			return discord.MessageID(ch.ID)
		}

		slices.SortFunc(channels, func(a, b discord.Channel) int {
			// Descending order
			return cmp.Compare(msgID(b), msgID(a))
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
		return tcell.NewEventKey(tcell.KeyRune, "K", tcell.ModNone)

	case gt.cfg.Keys.GuildsTree.SelectPrevious:
		return tcell.NewEventKey(tcell.KeyUp, "", tcell.ModNone)
	case gt.cfg.Keys.GuildsTree.SelectNext:
		return tcell.NewEventKey(tcell.KeyDown, "", tcell.ModNone)
	case gt.cfg.Keys.GuildsTree.SelectFirst:
		gt.Move(gt.GetRowCount() * -1)
		// return tcell.NewEventKey(tcell.KeyHome, 0, tcell.ModNone)
	case gt.cfg.Keys.GuildsTree.SelectLast:
		gt.Move(gt.GetRowCount())
		// return tcell.NewEventKey(tcell.KeyEnd, 0, tcell.ModNone)

	case gt.cfg.Keys.GuildsTree.SelectCurrent:
		return tcell.NewEventKey(tcell.KeyEnter, "", tcell.ModNone)

	case gt.cfg.Keys.GuildsTree.YankID:
		gt.yankID()
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
		go clipboard.Write(clipboard.FmtText, []byte(id.String()))
	}
}

func (gt *guildsTree) SelectChannelID(channelID discord.ChannelID) {
	if gt.chatView.state == nil || gt.chatView.state.Cabinet == nil {
		return
	}

	// First, check if the channel node exists
	var channelNode *tview.TreeNode
	gt.GetRoot().Walk(func(node, _ *tview.TreeNode) bool {
		if node.GetReference() == channelID {
			channelNode = node
			return false
		}
		return true
	})

	if channelNode != nil {
		gt.SetCurrentNode(channelNode)
		gt.onSelected(channelNode)
		return
	}

	// If not found, it might be because the guild hasn't been expanded yet.
	// Find the channel from state to get the GuildID.
	channel, err := gt.chatView.state.Cabinet.Channel(channelID)
	if err != nil {
		slog.Error("failed to get channel", "err", err, "channel_id", channelID)
		return
	}

	if !channel.GuildID.IsValid() {
		// DM or GroupDM. These should be under "Direct Messages" node.
		var dmNode *tview.TreeNode
		gt.GetRoot().Walk(func(node, _ *tview.TreeNode) bool {
			// DM node usually has no reference (nil) and specific text, or we can tag it better.
			// In onReady, it's created as tview.NewTreeNode("Direct Messages").
			if node.GetReference() == nil && node.GetText() == "Direct Messages" {
				dmNode = node
				return false
			}
			return true
		})

		if dmNode != nil {
			// Expand/Select DM node to populate
			gt.onSelected(dmNode)
			dmNode.SetExpanded(true)

			// Now try to find channel again
			gt.GetRoot().Walk(func(node, _ *tview.TreeNode) bool {
				if node.GetReference() == channelID {
					channelNode = node
					return false
				}
				return true
			})
			if channelNode != nil {
				gt.SetCurrentNode(channelNode)
				gt.onSelected(channelNode)
			}
		}
		return
	}

	// It's a guild channel. Find the guild node.
	var guildNode *tview.TreeNode
	gt.GetRoot().Walk(func(node, _ *tview.TreeNode) bool {
		if node.GetReference() == channel.GuildID {
			guildNode = node
			return false
		}
		return true
	})

	if guildNode != nil {
		// Expand guild to populate channels
		if len(guildNode.GetChildren()) == 0 {
			gt.onSelected(guildNode)
		}
		guildNode.SetExpanded(true)

		// Now find channel node
		gt.GetRoot().Walk(func(node, _ *tview.TreeNode) bool {
			if node.GetReference() == channelID {
				channelNode = node
				return false
			}
			return true
		})

		if channelNode != nil {
			gt.SetCurrentNode(channelNode)
			gt.onSelected(channelNode)
		}
	}
}
