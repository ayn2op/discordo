package guildstree

import (
	"log/slog"
	"slices"

	"github.com/ayn2op/arikawa/v3/discord"
	"github.com/ayn2op/arikawa/v3/gateway"
	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/discordo/internal/ui"
	"github.com/ayn2op/ningen/v3"
	"github.com/ayn2op/ningen/v3/states/read"
	"github.com/ayn2op/tview"
	"github.com/ayn2op/tview/help"
	"github.com/ayn2op/tview/keybind"
	"github.com/ayn2op/tview/tree"
	"github.com/gdamore/tcell/v3"
)

type dmNode struct{}

type Model struct {
	*tree.Model

	cfg   *config.Config
	state *ningen.State

	// Fast-path indexes for frequent event handlers (read updates, picker
	// navigation). They mirror the current rendered tree and are rebuilt on
	// READY before nodes are added.
	guildNodeByID   map[discord.GuildID]*tree.Node
	channelNodeByID map[discord.ChannelID]*tree.Node
	dmRootNode      *tree.Node
}

func NewModel(cfg *config.Config, state *ningen.State) *Model {
	m := &Model{
		Model: tree.NewModel(),
		cfg:   cfg,
		state: state,

		guildNodeByID:   make(map[discord.GuildID]*tree.Node),
		channelNodeByID: make(map[discord.ChannelID]*tree.Node),
	}
	ui.ConfigureBox(m.Box, &cfg.Theme)
	m.
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
	m.SetKeybinds(tree.Keybinds{
		Up:           cfg.Keybinds.GuildsTree.SelectUp.Keybind,
		Down:         cfg.Keybinds.GuildsTree.SelectDown.Keybind,
		Top:          cfg.Keybinds.GuildsTree.SelectTop.Keybind,
		Bottom:       cfg.Keybinds.GuildsTree.SelectBottom.Keybind,
		MoveToParent: cfg.Keybinds.GuildsTree.MoveToParentNode.Keybind,
		Select:       cfg.Keybinds.GuildsTree.SelectCurrent.Keybind,
	})

	return m
}

func (m *Model) reset() *tree.Node {
	// Keep allocated map capacity; READY can rebuild often during reconnects.
	clear(m.guildNodeByID)
	clear(m.channelNodeByID)
	m.dmRootNode = tree.NewNode("Direct Messages").
		SetReference(dmNode{}).
		SetExpandable(true).
		SetExpanded(false)
	return m.Root().ClearChildren().AddChild(m.dmRootNode)
}

func (m *Model) rebuild(event *gateway.ReadyEvent) {
	root := m.reset()
	guildsInFolders := make(map[discord.GuildID]bool)
	for _, folder := range event.UserSettings.GuildFolders {
		for _, guildID := range folder.GuildIDs {
			guildsInFolders[guildID] = true
		}
	}

	guildsByID := make(map[discord.GuildID]*gateway.GuildCreateEvent, len(event.Guilds))
	for i := range event.Guilds {
		guildsByID[event.Guilds[i].ID] = &event.Guilds[i]
	}

	positions := event.UserSettings.GuildPositions
	if len(positions) == 0 {
		positions = make([]discord.GuildID, 0, len(event.Guilds))
		for _, guild := range event.Guilds {
			positions = append(positions, guild.ID)
		}
	}
	for _, guildID := range positions {
		if guild := guildsByID[guildID]; guild != nil && !guildsInFolders[guildID] {
			m.createGuildNode(root, guild.Guild)
		}
	}

	for _, folder := range event.UserSettings.GuildFolders {
		if folder.ID == 0 && len(folder.GuildIDs) == 1 {
			if guild := guildsByID[folder.GuildIDs[0]]; guild != nil {
				m.createGuildNode(root, guild.Guild)
			}
		} else {
			m.createFolderNode(folder, guildsByID)
		}
	}
	m.SetCurrentNode(root)
}

func (m *Model) refreshReadStyles(event *read.UpdateEvent) {
	if event.GuildID.IsValid() {
		if node := m.findNodeByReference(event.GuildID); node != nil {
			m.setNodeLineStyle(node, m.guildNodeStyle(event.GuildID))
		}
	}

	node := m.findNodeByReference(event.ChannelID)
	if node == nil {
		return
	}
	channel, err := m.state.Cabinet.Channel(event.ChannelID)
	if err != nil {
		indication := m.state.ChannelIsUnread(event.ChannelID, ningen.UnreadOpts{IncludeMutedCategories: true})
		m.setNodeLineStyle(node, unreadStyle(indication))
		return
	}
	m.setNodeLineStyle(node, m.channelNodeStyle(*channel))
}

func (m *Model) updateDMNodeStyle(userID discord.UserID) {
	channel, err := m.state.Cabinet.CreatePrivateChannel(userID)
	if err != nil {
		return
	}

	node, ok := m.channelNodeByID[channel.ID]
	if node == nil || !ok {
		return
	}
	m.setNodeLineStyle(node, m.channelNodeStyle(*channel))
}

func (m *Model) moveDMToFront(channelID discord.ChannelID) {
	if m.dmRootNode == nil {
		return
	}

	node := m.channelNodeByID[channelID]
	children := m.dmRootNode.Children()
	if index := slices.Index(children, node); index > 0 {
		copy(children[1:index+1], children[:index])
		children[0] = node
	}
}

func (m *Model) createFolderNode(folder gateway.GuildFolder, guildsByID map[discord.GuildID]*gateway.GuildCreateEvent) {
	name := "Folder"
	if folder.Name != "" {
		name = folder.Name
	}

	folderNode := tree.NewNode(name).SetExpanded(m.cfg.Theme.GuildsTree.AutoExpandFolders)
	if folder.Color != 0 {
		folderStyle := tcell.StyleDefault.Foreground(tcell.NewHexColor(int32(folder.Color)))
		m.setNodeLineStyle(folderNode, folderStyle)
	}
	m.Root().AddChild(folderNode)

	for _, guildID := range folder.GuildIDs {
		if guildEvent, ok := guildsByID[guildID]; ok {
			m.createGuildNode(folderNode, guildEvent.Guild)
		}
	}
}

func (m *Model) guildNodeStyle(guildID discord.GuildID) tcell.Style {
	indication := m.state.GuildIsUnread(guildID, ningen.GuildUnreadOpts{IncludeMutedCategories: true})
	return unreadStyle(indication)
}

func (m *Model) channelNodeStyle(channel discord.Channel) tcell.Style {
	unread := unreadStyle(m.state.ChannelIsUnread(channel.ID, ningen.UnreadOpts{IncludeMutedCategories: true}))
	if channel.Type != discord.DirectMessage || len(channel.DMRecipients) != 1 {
		return unread
	}

	recipient := channel.DMRecipients[0]
	presence, err := m.state.Cabinet.Presence(discord.NullGuildID, recipient.ID)
	if err != nil {
		return tview.MergeStyle(m.dmStatusStyle(discord.OfflineStatus), unread)
	}

	return tview.MergeStyle(m.dmStatusStyle(presence.Status), unread)
}

func (m *Model) dmStatusStyle(status discord.Status) tcell.Style {
	switch status {
	case discord.DoNotDisturbStatus:
		return m.cfg.Theme.GuildsTree.DNDStyle.Style
	case discord.IdleStatus:
		return m.cfg.Theme.GuildsTree.IdleStyle.Style
	case discord.OnlineStatus:
		return m.cfg.Theme.GuildsTree.OnlineStyle.Style
	default:
		return m.cfg.Theme.GuildsTree.OfflineStyle.Style
	}
}

func (m *Model) createGuildNode(parent *tree.Node, guild discord.Guild) {
	guildNode := tree.NewNode(guild.Name).
		SetReference(guild.ID).
		SetExpandable(true).
		SetExpanded(false).
		SetIndent(m.cfg.Sidebar.Indents.Guild)
	m.setNodeLineStyle(guildNode, m.guildNodeStyle(guild.ID))
	parent.AddChild(guildNode)
	m.guildNodeByID[guild.ID] = guildNode
}

func (m *Model) createChannelNode(parent *tree.Node, channel discord.Channel) {
	if channel.Type != discord.DirectMessage && channel.Type != discord.GroupDM && channel.Type != discord.GuildCategory && !m.state.HasPermissions(channel.ID, discord.PermissionViewChannel) {
		return
	}

	indents := m.cfg.Sidebar.Indents
	channelNode := tree.NewNode(ui.ChannelToString(channel, m.cfg.Icons, m.state)).SetReference(channel.ID)
	m.setNodeLineStyle(channelNode, m.channelNodeStyle(channel))
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
	m.channelNodeByID[channel.ID] = channelNode
}

func (m *Model) setNodeLineStyle(node *tree.Node, style tcell.Style) {
	line := node.Line()
	for i := range line {
		line[i].Style = style
	}
	node.SetLine(line)
}

func (m *Model) createChannelNodes(node *tree.Node, channels []discord.Channel) {
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
			m.createChannelNode(node, channel)
		}
	}

	for _, channel := range channels {
		if channel.Type == discord.GuildCategory {
			if _, ok := hasChildByParentID[channel.ID]; ok {
				m.createChannelNode(node, channel)
			}
		}
	}

	for _, channel := range channels {
		if channel.ParentID.IsValid() {
			// Parent categories are inserted earlier in this function, so this
			// lookup is O(1) and avoids per-channel subtree walks.
			parent := m.channelNodeByID[channel.ParentID]
			if parent != nil {
				m.createChannelNode(parent, channel)
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

func (m *Model) collapseParentNode(node *tree.Node) {
	path := m.GetPath(node)
	if len(path) < 3 {
		return
	}
	parent := path[len(path)-2]
	parent.Collapse()
	m.SetCurrentNode(parent)
}

func (m *Model) Update(msg tview.Msg) tview.Cmd {
	switch msg := msg.(type) {
	case *gateway.ReadyEvent:
		m.rebuild(msg)
		return nil
	case *gateway.MessageCreateEvent:
		if !msg.GuildID.IsValid() {
			m.moveDMToFront(msg.ChannelID)
		}
		return nil
	case *gateway.PresenceUpdateEvent:
		m.updateDMNodeStyle(msg.User.ID)
		return nil
	case *read.UpdateEvent:
		m.refreshReadStyles(msg)
		return nil

	case NavigateMsg:
		return m.navigate(msg.ChannelID)
	case tree.SelectedMsg:
		return m.selectNode(msg.Node)
	case tview.KeyMsg:
		switch {
		case keybind.Matches(msg, m.cfg.Keybinds.GuildsTree.CollapseAll.Keybind):
			for _, node := range m.Root().Children() {
				node.CollapseAll()
			}
			return nil
		case keybind.Matches(msg, m.cfg.Keybinds.GuildsTree.CollapseParentNode.Keybind):
			m.collapseParentNode(m.CurrentNode())
			return nil
		case keybind.Matches(msg, m.cfg.Keybinds.GuildsTree.YankID.Keybind):
			return yankID(m.CurrentNode())
		}
	}
	return m.Model.Update(msg)
}

func (m *Model) findNodeByReference(reference any) *tree.Node {
	switch ref := reference.(type) {
	case discord.GuildID:
		return m.guildNodeByID[ref]
	case discord.ChannelID:
		return m.channelNodeByID[ref]
	case dmNode:
		return m.dmRootNode
	default:
		// Fallback keeps this helper safe for non-indexed custom references.
		var found *tree.Node
		m.Root().Walk(func(node, _ *tree.Node) bool {
			if node.Reference() == reference {
				found = node
				return false
			}
			return true
		})
		return found
	}
}

func (m *Model) findNodeByChannelID(channelID discord.ChannelID) *tree.Node {
	channel, err := m.state.Cabinet.Channel(channelID)
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
	if parent := m.findNodeByReference(reference); parent != nil {
		if len(parent.Children()) == 0 {
			m.selectNode(parent)
		}
	}

	node := m.findNodeByReference(channelID)
	return node
}

func (m *Model) expandPathToNode(node *tree.Node) {
	if node == nil {
		return
	}
	for _, n := range m.GetPath(node) {
		n.Expand()
	}
}

var _ help.KeyMap = (*Model)(nil)

func (m *Model) selectCurrentKeybind() keybind.Keybind {
	selectCurrent := m.cfg.Keybinds.GuildsTree.SelectCurrent.Keybind
	selectHelp := selectCurrent.Help()
	selectDesc := selectHelp.Desc
	if node := m.CurrentNode(); node != nil {
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

func (m *Model) ShortHelp() []keybind.Keybind {
	cfg := m.cfg.Keybinds.GuildsTree
	shortHelp := []keybind.Keybind{cfg.SelectUp.Keybind, cfg.SelectDown.Keybind, m.selectCurrentKeybind()}
	if m.canCollapseParent(m.CurrentNode()) {
		shortHelp = append(shortHelp, cfg.CollapseParentNode.Keybind)
	}
	return shortHelp
}

func (m *Model) FullHelp() [][]keybind.Keybind {
	cfg := m.cfg.Keybinds.GuildsTree
	selectGroup := []keybind.Keybind{m.selectCurrentKeybind(), cfg.MoveToParentNode.Keybind}
	selectGroup = append(selectGroup, m.collapseKeybinds()...)

	return [][]keybind.Keybind{
		{cfg.SelectUp.Keybind, cfg.SelectDown.Keybind, cfg.SelectTop.Keybind, cfg.SelectBottom.Keybind},
		selectGroup,
		{cfg.YankID.Keybind},
	}
}

func (m *Model) collapseKeybinds() []keybind.Keybind {
	cfg := m.cfg.Keybinds.GuildsTree

	var keybinds []keybind.Keybind
	if m.canCollapseParent(m.CurrentNode()) {
		keybinds = append(keybinds, cfg.CollapseParentNode.Keybind)
	}
	if m.canCollapseAll() {
		keybinds = append(keybinds, cfg.CollapseAll.Keybind)
	}
	return keybinds
}

func (m *Model) canCollapseParent(node *tree.Node) bool {
	return node != nil && len(m.GetPath(node)) >= 3
}

func (m *Model) canCollapseAll() bool {
	var can bool
	for _, node := range m.Root().Children() {
		if node.Expanded() {
			can = true
			break
		}
	}
	return can
}

func unreadStyle(indication ningen.UnreadIndication) tcell.Style {
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
