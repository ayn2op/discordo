package chat

import (
	"fmt"
	"log/slog"

	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/discordo/internal/ui"
	"github.com/ayn2op/tview"
	"github.com/ayn2op/tview/help"
	"github.com/ayn2op/tview/keybind"
	"github.com/ayn2op/tview/tree"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/diamondburned/ningen/v3"
	"github.com/gdamore/tcell/v3"
	"golang.design/x/clipboard"
)

type dmNode struct{}

type guildsTree struct {
	*tree.Model
	chat *Model

	cfg *config.Config

	// Fast-path indexes for frequent event handlers (read updates, picker
	// navigation). They mirror the current rendered tree and are rebuilt on
	// READY before nodes are added.
	guildNodeByID   map[discord.GuildID]*tree.Node
	channelNodeByID map[discord.ChannelID]*tree.Node
	dmRootNode      *tree.Node
}

func newGuildsTree(cfg *config.Config, chat *Model) *guildsTree {
	gt := &guildsTree{
		Model: tree.NewModel(),
		cfg:   cfg,
		chat:  chat,

		guildNodeByID:   make(map[discord.GuildID]*tree.Node),
		channelNodeByID: make(map[discord.ChannelID]*tree.Node),
	}

	gt.Box = ui.ConfigureBox(gt.Box, &cfg.Theme)
	gt.
		SetRoot(tree.NewNode("")).
		SetTopLevel(1).
		SetMarkers(tree.Markers{
			Expanded:  cfg.Sidebar.Markers.Expanded,
			Collapsed: cfg.Sidebar.Markers.Collapsed,
			Leaf:      cfg.Sidebar.Markers.Leaf,
		}).
		SetGraphics(cfg.Theme.GuildsTree.Graphics).
		SetGraphicsColor(tcell.GetColor(cfg.Theme.GuildsTree.GraphicsColor)).
		SetTitle("Guilds")
	gt.SetKeybinds(tree.Keybinds{
		Up:           cfg.Keybinds.GuildsTree.SelectUp.Keybind,
		Down:         cfg.Keybinds.GuildsTree.SelectDown.Keybind,
		Top:          cfg.Keybinds.GuildsTree.SelectTop.Keybind,
		Bottom:       cfg.Keybinds.GuildsTree.SelectBottom.Keybind,
		MoveToParent: cfg.Keybinds.GuildsTree.MoveToParentNode.Keybind,
		Select:       cfg.Keybinds.GuildsTree.SelectCurrent.Keybind,
	})

	return gt
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

	folderNode := tree.NewNode(name).SetExpanded(gt.cfg.Theme.GuildsTree.AutoExpandFolders)
	if folder.Color != 0 {
		folderStyle := tcell.StyleDefault.Foreground(tcell.NewHexColor(int32(folder.Color)))
		gt.setNodeLineStyle(folderNode, folderStyle)
	}
	gt.Root().AddChild(folderNode)

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

func (gt *guildsTree) guildNodeStyle(guildID discord.GuildID) tcell.Style {
	indication := gt.chat.state.GuildIsUnread(guildID, ningen.GuildUnreadOpts{UnreadOpts: ningen.UnreadOpts{IncludeMutedCategories: true}})
	return gt.unreadStyle(indication)
}

func (gt *guildsTree) channelNodeStyle(channel discord.Channel) tcell.Style {
	unread := gt.unreadStyle(gt.chat.state.ChannelIsUnread(channel.ID, ningen.UnreadOpts{IncludeMutedCategories: true}))
	if channel.Type != discord.DirectMessage || len(channel.DMRecipients) != 1 {
		return unread
	}

	recipient := channel.DMRecipients[0]
	presence, err := gt.chat.state.Cabinet.Presence(discord.NullGuildID, recipient.ID)
	if err != nil {
		return tview.MergeStyle(gt.dmStatusStyle(discord.OfflineStatus), unread)
	}

	return tview.MergeStyle(gt.dmStatusStyle(presence.Status), unread)
}

func (gt *guildsTree) dmStatusStyle(status discord.Status) tcell.Style {
	switch status {
	case discord.DoNotDisturbStatus:
		return gt.cfg.Theme.GuildsTree.DNDStyle.Style
	case discord.IdleStatus:
		return gt.cfg.Theme.GuildsTree.IdleStyle.Style
	case discord.OnlineStatus:
		return gt.cfg.Theme.GuildsTree.OnlineStyle.Style
	default:
		return gt.cfg.Theme.GuildsTree.OfflineStyle.Style
	}
}

func (gt *guildsTree) createGuildNode(parent *tree.Node, guild discord.Guild) {
	guildNode := tree.NewNode(guild.Name).
		SetReference(guild.ID).
		SetExpandable(true).
		SetExpanded(false).
		SetIndent(gt.cfg.Sidebar.Indents.Guild)
	gt.setNodeLineStyle(guildNode, gt.guildNodeStyle(guild.ID))
	parent.AddChild(guildNode)
	gt.guildNodeByID[guild.ID] = guildNode
}

func (gt *guildsTree) createChannelNode(parent *tree.Node, channel discord.Channel) {
	if channel.Type != discord.DirectMessage && channel.Type != discord.GroupDM && channel.Type != discord.GuildCategory && !gt.chat.state.HasPermissions(channel.ID, discord.PermissionViewChannel) {
		return
	}

	indents := gt.cfg.Sidebar.Indents
	channelNode := tree.NewNode(ui.ChannelToString(channel, gt.cfg.Icons, gt.chat.state)).SetReference(channel.ID)
	gt.setNodeLineStyle(channelNode, gt.channelNodeStyle(channel))
	switch channel.Type {
	case discord.DirectMessage:
		channelNode.SetIndent(indents.DM)
	case discord.GroupDM:
		channelNode.SetIndent(indents.GroupDM)
	case discord.GuildCategory:
		channelNode.SetIndent(indents.Category)
		channelNode.SetExpandable(true).SetExpanded(true)
	case discord.GuildForum:
		channelNode.SetIndent(indents.Forum)
		channelNode.SetExpandable(true).SetExpanded(false)
	default:
		channelNode.SetIndent(indents.Channel)
	}
	parent.AddChild(channelNode)
	gt.channelNodeByID[channel.ID] = channelNode
}

func (gt *guildsTree) setNodeLineStyle(node *tree.Node, style tcell.Style) {
	line := node.Line()
	for i := range line {
		line[i].Style = style
	}
	node.SetLine(line)
}

func (gt *guildsTree) createChannelNodes(node *tree.Node, channels []discord.Channel) {
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

func isThread(t discord.ChannelType) bool {
	switch t {
	case discord.GuildPublicThread, discord.GuildPrivateThread, discord.GuildAnnouncementThread:
		return true
	default:
		return false
	}
}

func (gt *guildsTree) onSelected(node *tree.Node) tview.Cmd {
	if len(node.Children()) != 0 {
		node.SetExpanded(!node.Expanded())
		return nil
	}

	switch ref := node.Reference().(type) {
	case discord.GuildID:
		go gt.chat.state.MemberState.Subscribe(ref)

		channels, err := gt.chat.state.Cabinet.Channels(ref)
		if err != nil {
			slog.Error("failed to get channels", "err", err, "guild_id", ref)
			return nil
		}

		ui.SortGuildChannels(channels)
		gt.createChannelNodes(node, channels)
		node.Expand()
		return nil
	case discord.ChannelID:
		channel, err := gt.chat.state.Cabinet.Channel(ref)
		if err != nil {
			slog.Error("failed to get channel from state", "err", err, "channel_id", ref)
			return nil
		}

		// Forums contain threads, not messages; load the threads as children.
		if channel.Type == discord.GuildForum {
			allChannels, err := gt.chat.state.Cabinet.Channels(channel.GuildID)
			if err != nil {
				slog.Error("failed to get channels for forum threads", "err", err, "guild_id", channel.GuildID)
				return nil
			}

			for _, ch := range allChannels {
				if ch.ParentID == channel.ID && isThread(ch.Type) {
					gt.createChannelNode(node, ch)
				}
			}
			node.Expand()
			return nil
		}

		return gt.loadChannel(*channel)
	case dmNode: // Direct messages folder
		channels, err := gt.chat.state.PrivateChannels()
		if err != nil {
			slog.Error("failed to get private channels", "err", err)
			return nil
		}

		ui.SortPrivateChannels(channels)
		for _, c := range channels {
			gt.createChannelNode(node, c)
		}
		node.Expand()
		return nil
	}
	return nil
}

func (gt *guildsTree) loadChannel(channel discord.Channel) tview.Cmd {
	limit := uint(gt.cfg.MessagesLimit)
	return func() tview.Msg {
		messages, err := gt.chat.state.Messages(channel.ID, limit)
		if err != nil {
			slog.Error("failed to get messages", "err", err, "channel_id", channel.ID, "limit", limit)
			return nil
		}

		go gt.chat.state.ReadState.MarkRead(channel.ID, channel.LastMessageID)

		if guildID := channel.GuildID; guildID.IsValid() {
			gt.chat.messagesList.requestGuildMembers(guildID, messages)
		}
		return channelLoadedMsg{Channel: channel, Messages: messages}
	}
}

func (gt *guildsTree) collapseParentNode(node *tree.Node) {
	gt.
		Root().
		Walk(func(n, parent *tree.Node) bool {
			if n == node && parent.GetLevel() != 0 {
				parent.Collapse()
				gt.SetCurrentNode(parent)
				return false
			}

			return true
		})
}

func (gt *guildsTree) Update(msg tview.Msg) tview.Cmd {
	switch msg := msg.(type) {
	case tree.SelectedMsg:
		return gt.onSelected(msg.Node)
	case tview.KeyMsg:
		switch {
		case keybind.Matches(msg, gt.cfg.Keybinds.GuildsTree.CollapseAll.Keybind):
			for _, node := range gt.Root().Children() {
				node.CollapseAll()
			}
			return nil
		case keybind.Matches(msg, gt.cfg.Keybinds.GuildsTree.CollapseParentNode.Keybind):
			gt.collapseParentNode(gt.CurrentNode())
			return nil
		case keybind.Matches(msg, gt.cfg.Keybinds.GuildsTree.YankID.Keybind):
			return gt.yankID()
		}
	}
	return gt.Model.Update(msg)
}

func (gt *guildsTree) yankID() tview.Cmd {
	node := gt.CurrentNode()
	if node == nil {
		return nil
	}

	// Reference of a tree node in the guilds tree is its ID.
	// discord.Snowflake (discord.GuildID and discord.ChannelID) have the String method.
	if id, ok := node.Reference().(fmt.Stringer); ok {
		return func() tview.Msg {
			if err := clipboard.Write(clipboard.FmtText, []byte(id.String())); err != nil {
				slog.Error("failed to copy node id", "err", err)
			}
			return nil
		}
	}
	return nil
}

func (gt *guildsTree) findNodeByReference(reference any) *tree.Node {
	switch ref := reference.(type) {
	case discord.GuildID:
		return gt.guildNodeByID[ref]
	case discord.ChannelID:
		return gt.channelNodeByID[ref]
	case dmNode:
		return gt.dmRootNode
	default:
		// Fallback keeps this helper safe for non-indexed custom references.
		var found *tree.Node
		gt.Root().Walk(func(node, _ *tree.Node) bool {
			if node.Reference() == reference {
				found = node
				return false
			}
			return true
		})
		return found
	}
}

func (gt *guildsTree) findNodeByChannelID(channelID discord.ChannelID) *tree.Node {
	channel, err := gt.chat.state.Cabinet.Channel(channelID)
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
	if parent := gt.findNodeByReference(reference); parent != nil {
		if len(parent.Children()) == 0 {
			gt.onSelected(parent)
		}
	}

	node := gt.findNodeByReference(channelID)
	return node
}

func (gt *guildsTree) expandPathToNode(node *tree.Node) {
	if node == nil {
		return
	}
	for _, n := range gt.GetPath(node) {
		n.Expand()
	}
}

var _ help.KeyMap = (*guildsTree)(nil)

func (gt *guildsTree) selectCurrentKeybind() keybind.Keybind {
	selectCurrent := gt.cfg.Keybinds.GuildsTree.SelectCurrent.Keybind
	selectHelp := selectCurrent.Help()
	selectDesc := selectHelp.Desc
	if node := gt.CurrentNode(); node != nil {
		if len(node.Children()) > 0 {
			if node.Expanded() {
				selectDesc = "collapse"
			} else {
				selectDesc = "expand"
			}
		} else {
			switch node.Reference().(type) {
			case discord.GuildID, dmNode:
				selectDesc = "expand"
			}
		}
	}
	selectCurrent.SetHelp(selectHelp.Key, selectDesc)
	return selectCurrent
}

func (gt *guildsTree) ShortHelp() []keybind.Keybind {
	cfg := gt.cfg.Keybinds.GuildsTree
	shortHelp := []keybind.Keybind{cfg.SelectUp.Keybind, cfg.SelectDown.Keybind, gt.selectCurrentKeybind()}
	if gt.canCollapseParent(gt.CurrentNode()) {
		shortHelp = append(shortHelp, cfg.CollapseParentNode.Keybind)
	}
	return shortHelp
}

func (gt *guildsTree) FullHelp() [][]keybind.Keybind {
	cfg := gt.cfg.Keybinds.GuildsTree
	selectGroup := []keybind.Keybind{gt.selectCurrentKeybind(), cfg.MoveToParentNode.Keybind}
	selectGroup = append(selectGroup, gt.collapseKeybinds()...)

	return [][]keybind.Keybind{
		{cfg.SelectUp.Keybind, cfg.SelectDown.Keybind, cfg.SelectTop.Keybind, cfg.SelectBottom.Keybind},
		selectGroup,
		{cfg.YankID.Keybind},
	}
}

func (gt *guildsTree) collapseKeybinds() []keybind.Keybind {
	cfg := gt.cfg.Keybinds.GuildsTree

	var keybinds []keybind.Keybind
	if gt.canCollapseParent(gt.CurrentNode()) {
		keybinds = append(keybinds, cfg.CollapseParentNode.Keybind)
	}
	if gt.canCollapseAll() {
		keybinds = append(keybinds, cfg.CollapseAll.Keybind)
	}
	return keybinds
}

func (gt *guildsTree) canCollapseParent(node *tree.Node) bool {
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

func (gt *guildsTree) canCollapseAll() bool {
	var can bool
	for _, node := range gt.Root().Children() {
		if node.Expanded() {
			can = true
			break
		}
	}
	return can
}
