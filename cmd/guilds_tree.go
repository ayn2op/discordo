package cmd

import (
	"fmt"
	"log/slog"
	"sort"

	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/discordo/internal/ui"
	"github.com/ayn2op/tview"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/diamondburned/ningen/v3"
	"github.com/gdamore/tcell/v2"
	"golang.design/x/clipboard"
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
		name = fmt.Sprintf("[%s]%s[-]", folder.Color, folder.Name)
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
	indication := discordState.GuildIsUnread(guildID, ningen.GuildUnreadOpts{UnreadOpts: ningen.UnreadOpts{IncludeMutedCategories: true}})
	return gt.unreadStyle(indication)
}

func (gt *guildsTree) getChannelNodeStyle(channelID discord.ChannelID) tcell.Style {
	indication := discordState.ChannelIsUnread(channelID, ningen.UnreadOpts{IncludeMutedCategories: true})
	return gt.unreadStyle(indication)
}

func (gt *guildsTree) createGuildNode(n *tview.TreeNode, guild discord.Guild) {
	guildNode := tview.NewTreeNode(guild.Name).
		SetReference(guild.ID).
		SetTextStyle(gt.getGuildNodeStyle(guild.ID))
	n.AddChild(guildNode)
}

func (gt *guildsTree) createChannelNode(node *tview.TreeNode, channel discord.Channel) {
	if channel.Type != discord.DirectMessage && channel.Type != discord.GroupDM && !discordState.HasPermissions(channel.ID, discord.PermissionViewChannel) {
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

		go discordState.ReadState.MarkRead(channel.ID, channel.LastMessageID)

		messages, err := discordState.Messages(channel.ID, uint(gt.cfg.MessagesLimit))
		if err != nil {
			slog.Error("failed to get messages", "err", err, "channel_id", channel.ID, "limit", gt.cfg.MessagesLimit)
			return
		}

		if guildID := channel.GuildID; guildID.IsValid() {
			app.messagesList.requestGuildMembers(guildID, messages)
		}

		app.messagesList.reset()
		app.messagesList.setTitle(*channel)
		app.messagesList.drawMessages(messages)
		app.messagesList.ScrollToEnd()

		hasNoPerm := channel.Type != discord.DirectMessage && channel.Type != discord.GroupDM && !discordState.HasPermissions(channel.ID, discord.PermissionSendMessages)
		app.messageInput.SetDisabled(hasNoPerm)
		if hasNoPerm {
			app.messageInput.SetPlaceholder("You do not have permission to send messages in this channel.")
		} else {
			app.messageInput.SetPlaceholder("Message...")
			app.SetFocus(app.messageInput)
		}

		gt.selectedChannelID = channel.ID
		gt.selectedGuildID = channel.GuildID
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
