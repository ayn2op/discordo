package chat

import (
	"fmt"
	"log/slog"

	"github.com/ayn2op/discordo/internal/clipboard"
	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/discordo/internal/ui"
	"github.com/ayn2op/tview"
	"github.com/ayn2op/tview/help"
	"github.com/ayn2op/tview/keybind"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/diamondburned/ningen/v3"
	"github.com/gdamore/tcell/v3"
)

type dmNode struct{}

type guildsTree struct {
	*tview.TreeView
	cfg      *config.Config
	chatView *View

	// Fast-path indexes for frequent event handlers (read updates, picker
	// navigation). They mirror the current rendered tree and are rebuilt on
	// READY before nodes are added.
	guildNodeByID   map[discord.GuildID]*tview.TreeNode
	channelNodeByID map[discord.ChannelID]*tview.TreeNode
	dmRootNode      *tview.TreeNode
}

var _ help.KeyMap = (*guildsTree)(nil)

func newGuildsTree(cfg *config.Config, chatView *View) *guildsTree {
	gt := &guildsTree{
		TreeView: tview.NewTreeView(),
		cfg:      cfg,
		chatView: chatView,

		guildNodeByID:   make(map[discord.GuildID]*tview.TreeNode),
		channelNodeByID: make(map[discord.ChannelID]*tview.TreeNode),
	}

	gt.Box = ui.ConfigureBox(gt.Box, &cfg.Theme)
	gt.
		SetRoot(tview.NewTreeNode("")).
		SetTopLevel(1).
		SetMarkers(tview.TreeMarkers{
			Expanded:  cfg.Sidebar.Markers.Expanded,
			Collapsed: cfg.Sidebar.Markers.Collapsed,
			Leaf:      cfg.Sidebar.Markers.Leaf,
		}).
		SetGraphics(cfg.Theme.GuildsTree.Graphics).
		SetGraphicsColor(tcell.GetColor(cfg.Theme.GuildsTree.GraphicsColor)).
		SetSelectedFunc(gt.onSelected).
		SetTitle("Guilds")

	return gt
}

func (gt *guildsTree) ShortHelp() []keybind.Keybind {
	cfg := gt.cfg.Keybinds.GuildsTree
	selectCurrent := cfg.SelectCurrent.Keybind
	collapseParent := cfg.CollapseParentNode.Keybind
	selectHelp := selectCurrent.Help()
	selectDesc := selectHelp.Desc
	if node := gt.GetCurrentNode(); node != nil {
		if len(node.GetChildren()) > 0 {
			if node.IsExpanded() {
				selectDesc = "collapse"
			} else {
				selectDesc = "expand"
			}
		} else {
			switch node.GetReference().(type) {
			case discord.GuildID, dmNode:
				selectDesc = "expand"
			}
		}
	}
	selectCurrent.SetHelp(selectHelp.Key, selectDesc)
	collapseHelp := collapseParent.Help()
	collapseParent.SetHelp(collapseHelp.Key, "collapse parent")

	shortHelp := []keybind.Keybind{cfg.Up.Keybind, cfg.Down.Keybind, selectCurrent}
	if gt.canCollapseParent(gt.GetCurrentNode()) {
		shortHelp = append(shortHelp, collapseParent)
	}
	return shortHelp
}

func (gt *guildsTree) FullHelp() [][]keybind.Keybind {
	cfg := gt.cfg.Keybinds.GuildsTree
	selectCurrent := cfg.SelectCurrent.Keybind
	collapseParent := cfg.CollapseParentNode.Keybind
	selectHelp := selectCurrent.Help()
	selectDesc := selectHelp.Desc
	if node := gt.GetCurrentNode(); node != nil {
		if len(node.GetChildren()) > 0 {
			if node.IsExpanded() {
				selectDesc = "collapse"
			} else {
				selectDesc = "expand"
			}
		} else {
			switch node.GetReference().(type) {
			case discord.GuildID, dmNode:
				selectDesc = "expand"
			}
		}
	}
	selectCurrent.SetHelp(selectHelp.Key, selectDesc)
	collapseHelp := collapseParent.Help()
	collapseParent.SetHelp(collapseHelp.Key, "collapse parent")

	actions := []keybind.Keybind{selectCurrent, cfg.MoveToParentNode.Keybind}
	if gt.canCollapseParent(gt.GetCurrentNode()) {
		actions = append(actions, collapseParent)
	}

	return [][]keybind.Keybind{
		{cfg.Up.Keybind, cfg.Down.Keybind, cfg.Top.Keybind, cfg.Bottom.Keybind},
		actions,
		{cfg.YankID.Keybind},
	}
}

func (gt *guildsTree) canCollapseParent(node *tview.TreeNode) bool {
	if node == nil {
		return false
	}
	path := gt.GetPath(node)
	// Path layout is [root, ..., node]. A non-root parent means at least 3 nodes.
	if len(path) < 3 {
		return false
	}
	parent := path[len(path)-2]
	return parent != nil && parent.GetLevel() != 0
}

func (gt *guildsTree) resetNodeIndex() {
	// Keep allocated map capacity; READY can rebuild often during reconnects.
	clear(gt.guildNodeByID)
	clear(gt.channelNodeByID)
	gt.dmRootNode = nil
}

func (gt *guildsTree) createFolderNode(folder gateway.GuildFolder, guildsByID map[discord.GuildID]*gateway.GuildCreateEvent) {
	name := "Folder"
	if folder.Name != "" {
		name = folder.Name
	}

	folderNode := tview.NewTreeNode(name).SetExpanded(gt.cfg.Theme.GuildsTree.AutoExpandFolders)
	if folder.Color != 0 {
		folderStyle := tcell.StyleDefault.Foreground(tcell.NewHexColor(int32(folder.Color)))
		gt.setNodeLineStyle(folderNode, folderStyle)
	}
	gt.GetRoot().AddChild(folderNode)

	for _, guildID := range folder.GuildIDs {
		if guildEvent, ok := guildsByID[guildID]; ok {
			gt.createGuildNode(folderNode, guildEvent.Guild)
		}
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
	guildNode := tview.NewTreeNode(guild.Name).SetReference(guild.ID).SetExpandable(true).SetExpanded(false)
	gt.setNodeLineStyle(guildNode, gt.getGuildNodeStyle(guild.ID))
	n.AddChild(guildNode)
	gt.guildNodeByID[guild.ID] = guildNode
}

func (gt *guildsTree) createChannelNode(node *tview.TreeNode, channel discord.Channel) {
	if channel.Type != discord.DirectMessage && channel.Type != discord.GroupDM && channel.Type != discord.GuildCategory && !gt.chatView.state.HasPermissions(channel.ID, discord.PermissionViewChannel) {
		return
	}

	channelNode := tview.NewTreeNode(ui.ChannelToString(channel, gt.cfg.Icons, gt.chatView.state)).SetReference(channel.ID)
	if channel.Type == discord.GuildForum {
		channelNode.SetExpandable(true).SetExpanded(false)
	}
	gt.setNodeLineStyle(channelNode, gt.getChannelNodeStyle(channel.ID))
	node.AddChild(channelNode)
	gt.channelNodeByID[channel.ID] = channelNode
}

func (gt *guildsTree) setNodeLineStyle(node *tview.TreeNode, style tcell.Style) {
	line := node.GetLine()
	for i := range line {
		line[i].Style = style
	}
	node.SetLine(line)
}

func (gt *guildsTree) createChannelNodes(node *tview.TreeNode, channels []discord.Channel) {
	// Preserve exact ordering semantics:
	// 1) top-level non-categories (in input order),
	// 2) categories that have at least one child in the source slice (in input order),
	// 3) parented channels under already-created categories (in input order).
	//
	// We precompute parent presence once to avoid the O(n^2) category-child scan.
	hasChildByParentID := make(map[discord.ChannelID]struct{}, len(channels))
	for _, channel := range channels {
		if channel.ParentID.IsValid() {
			hasChildByParentID[channel.ParentID] = struct{}{}
		}
	}

	for _, channel := range channels {
		if channel.Type != discord.GuildCategory && !channel.ParentID.IsValid() {
			gt.createChannelNode(node, channel)
		}
	}

	for _, channel := range channels {
		if channel.Type == discord.GuildCategory {
			if _, ok := hasChildByParentID[channel.ID]; ok {
				gt.createChannelNode(node, channel)
			}
		}
	}

	for _, channel := range channels {
		if channel.ParentID.IsValid() {
			// Parent categories are inserted earlier in this function, so this
			// lookup is O(1) and avoids per-channel subtree walks.
			parent := gt.channelNodeByID[channel.ParentID]
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

		ui.SortGuildChannels(channels)
		gt.createChannelNodes(node, channels)
		node.Expand()
	case discord.ChannelID:
		channel, err := gt.chatView.state.Cabinet.Channel(ref)
		if err != nil {
			slog.Error("failed to get channel from state", "channel_id", ref)
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
			node.Expand()
			return
		}

		limit := gt.cfg.MessagesLimit
		messages, err := gt.chatView.state.Messages(channel.ID, uint(limit))
		if err != nil {
			slog.Error("failed to get messages", "err", err, "channel_id", channel.ID, "limit", limit)
			return
		}

		go gt.chatView.state.ReadState.MarkRead(channel.ID, channel.LastMessageID)

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
		var text string
		if hasNoPerm {
			text = "You do not have permission to send messages in this channel."
		} else {
			text = "Message..."
			if gt.cfg.AutoFocus {
				gt.chatView.app.SetFocus(gt.chatView.messageInput)
			}
		}
		gt.chatView.messageInput.SetPlaceholder(tview.NewLine(tview.NewSegment(text, tcell.StyleDefault.Dim(true))))
	case dmNode: // Direct messages folder
		channels, err := gt.chatView.state.PrivateChannels()
		if err != nil {
			slog.Error("failed to get private channels", "err", err)
			return
		}

		ui.SortPrivateChannels(channels)
		for _, c := range channels {
			gt.createChannelNode(node, c)
		}
		node.Expand()
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

func (gt *guildsTree) handleInput(event *tcell.EventKey) *tcell.EventKey {
	switch {
	case keybind.Matches(event, gt.cfg.Keybinds.GuildsTree.CollapseParentNode.Keybind):
		gt.collapseParentNode(gt.GetCurrentNode())
		return nil
	case keybind.Matches(event, gt.cfg.Keybinds.GuildsTree.MoveToParentNode.Keybind):
		return tcell.NewEventKey(tcell.KeyRune, "K", tcell.ModNone)

	case keybind.Matches(event, gt.cfg.Keybinds.GuildsTree.Up.Keybind):
		return tcell.NewEventKey(tcell.KeyUp, "", tcell.ModNone)
	case keybind.Matches(event, gt.cfg.Keybinds.GuildsTree.Down.Keybind):
		return tcell.NewEventKey(tcell.KeyDown, "", tcell.ModNone)
	case keybind.Matches(event, gt.cfg.Keybinds.GuildsTree.Top.Keybind):
		gt.Move(gt.GetRowCount() * -1)
		// return tcell.NewEventKey(tcell.KeyHome, 0, tcell.ModNone)
	case keybind.Matches(event, gt.cfg.Keybinds.GuildsTree.Bottom.Keybind):
		gt.Move(gt.GetRowCount())
		// return tcell.NewEventKey(tcell.KeyEnd, 0, tcell.ModNone)

	case keybind.Matches(event, gt.cfg.Keybinds.GuildsTree.SelectCurrent.Keybind):
		return tcell.NewEventKey(tcell.KeyEnter, "", tcell.ModNone)

	case keybind.Matches(event, gt.cfg.Keybinds.GuildsTree.YankID.Keybind):
		gt.yankID()
	}

	return nil
}

func (gt *guildsTree) InputHandler(event *tcell.EventKey, setFocus func(p tview.Primitive)) {
	event = gt.handleInput(event)
	if event == nil {
		return
	}
	gt.TreeView.InputHandler(event, setFocus)
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

func (gt *guildsTree) findNodeByReference(reference any) *tview.TreeNode {
	switch ref := reference.(type) {
	case discord.GuildID:
		return gt.guildNodeByID[ref]
	case discord.ChannelID:
		return gt.channelNodeByID[ref]
	case dmNode:
		return gt.dmRootNode
	default:
		// Fallback keeps this helper safe for non-indexed custom references.
		var found *tview.TreeNode
		gt.GetRoot().Walk(func(node, _ *tview.TreeNode) bool {
			if node.GetReference() == reference {
				found = node
				return false
			}
			return true
		})
		return found
	}
}

func (gt *guildsTree) findNodeByChannelID(channelID discord.ChannelID) *tview.TreeNode {
	channel, err := gt.chatView.state.Cabinet.Channel(channelID)
	if err != nil {
		slog.Error("failed to get channel", "channel_id", channelID, "err", err)
		return nil
	}

	var reference any
	if guildID := channel.GuildID; guildID.IsValid() {
		reference = guildID
	} else {
		reference = dmNode{}
	}
	if parentNode := gt.findNodeByReference(reference); parentNode != nil {
		if len(parentNode.GetChildren()) == 0 {
			gt.onSelected(parentNode)
		}
	}

	node := gt.findNodeByReference(channelID)
	return node
}

func (gt *guildsTree) expandPathToNode(node *tview.TreeNode) {
	if node == nil {
		return
	}
	for _, n := range gt.GetPath(node) {
		n.Expand()
	}
}
