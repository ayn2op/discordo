package guildstree

import (
	"cmp"
	"log/slog"
	"slices"
	"time"

	"github.com/ayn2op/arikawa/v3/discord"
	"github.com/ayn2op/arikawa/v3/gateway"
	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/discordo/internal/ui"
	uitree "github.com/ayn2op/discordo/internal/ui/tree"
	"github.com/ayn2op/discordo/internal/voice"
	"github.com/ayn2op/ningen/v3"
	"github.com/ayn2op/ningen/v3/states/read"
	"github.com/ayn2op/tview"
	"github.com/ayn2op/tview/keybind"
	"github.com/ayn2op/tview/tree"
	"github.com/gdamore/tcell/v3"
)

type dmNode struct{}

type Model struct {
	root           *tree.Node
	selectionState tree.SelectionState

	cfg   *config.Config
	state *ningen.State
	voice *voice.Session

	// nodes indexes the guild and channel nodes for frequent event handlers (read updates, picker navigation).
	// It mirrors the current rendered tree and is rebuilt on READY before nodes are added.
	nodes      map[discord.Snowflake]*tree.Node
	dmRootNode *tree.Node
}

func NewModel(cfg *config.Config, state *ningen.State) Model {
	return Model{
		root:  tree.NewNode(""),
		cfg:   cfg,
		state: state,
		voice: voice.New(state.State, cfg.Voice.Sensitivity),
		nodes: make(map[discord.Snowflake]*tree.Node),
	}
}

// CurrentNode returns the selected node, or nil for none.
func (m Model) CurrentNode() *tree.Node {
	return m.selectionState.CurrentNode()
}

// View shows the tree in a box titled Guilds.
func (m Model) View(focused bool) tview.Widget {
	onChange := func(c tree.Change) tview.Msg { return Msg(c) }
	onSelect := func(n *tree.Node) tview.Msg { return SelectedMsg{Node: n} }
	return uitree.New(
		m.root,
		m.selectionState,
		m.cfg,
		m.cfg.Theme.GuildsTree.TreeThemeConfig,
		m.cfg.UI.GuildsTree.Graphics,
		m.cfg.Keybinds.GuildsTree.TreeKeybinds,
		focused,
		onChange,
		onSelect,
	).Title("Guilds").Footer(m.voiceFooter())
}

func (m *Model) reset() *tree.Node {
	// Keep allocated map capacity; READY can rebuild often during reconnects.
	clear(m.nodes)
	m.dmRootNode = tree.NewNode("Direct Messages").
		SetReference(dmNode{}).
		SetExpandable(true).
		SetExpanded(false)
	return m.root.ClearChildren().AddChild(m.dmRootNode)
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
	m.selectionState.SetCurrentNode(root)
}

func (m Model) refreshReadStyles(event *read.UpdateEvent) {
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

func (m Model) updateDMNodeStyle(userID discord.UserID) {
	channel, err := m.state.Cabinet.CreatePrivateChannel(userID)
	if err != nil {
		return
	}

	node, ok := m.nodes[discord.Snowflake(channel.ID)]
	if node == nil || !ok {
		return
	}
	m.setNodeLineStyle(node, m.channelNodeStyle(*channel))
}

func (m Model) moveDMToFront(channelID discord.ChannelID) {
	if m.dmRootNode == nil {
		return
	}

	node := m.nodes[discord.Snowflake(channelID)]
	children := m.dmRootNode.Children()
	if index := slices.Index(children, node); index > 0 {
		copy(children[1:index+1], children[:index])
		children[0] = node
	}
}

func (m Model) createFolderNode(folder gateway.GuildFolder, guildsByID map[discord.GuildID]*gateway.GuildCreateEvent) {
	name := "Folder"
	if folder.Name != "" {
		name = folder.Name
	}

	folderNode := tree.NewNode(name).SetExpanded(m.cfg.UI.GuildsTree.AutoExpandFolders)
	if folder.Color != 0 {
		folderStyle := tcell.StyleDefault.Foreground(tcell.NewHexColor(int32(folder.Color)))
		m.setNodeLineStyle(folderNode, folderStyle)
	}
	m.root.AddChild(folderNode)

	for _, guildID := range folder.GuildIDs {
		if guildEvent, ok := guildsByID[guildID]; ok {
			m.createGuildNode(folderNode, guildEvent.Guild)
		}
	}
}

func (m Model) guildNodeStyle(guildID discord.GuildID) tview.Style {
	indication := m.state.GuildIsUnread(guildID, ningen.GuildUnreadOpts{IncludeMutedCategories: true})
	return unreadStyle(indication)
}

func (m Model) channelNodeStyle(channel discord.Channel) tview.Style {
	unread := unreadStyle(m.state.ChannelIsUnread(channel.ID, ningen.UnreadOpts{IncludeMutedCategories: true}))
	if channel.Type != discord.DirectMessage || len(channel.DMRecipients) != 1 {
		return unread
	}

	recipient := channel.DMRecipients[0]
	presence, err := m.state.Cabinet.Presence(discord.NullGuildID, recipient.ID)
	if err != nil {
		return tview.MergeStyle(m.cfg.Theme.GuildsTree.StatusStyle(discord.OfflineStatus), unread)
	}

	return tview.MergeStyle(m.cfg.Theme.GuildsTree.StatusStyle(presence.Status), unread)
}

func (m Model) createGuildNode(parent *tree.Node, guild discord.Guild) {
	guildNode := tree.NewNode(guild.Name).
		SetReference(guild.ID).
		SetExpandable(true).
		SetExpanded(false).
		SetIndent(m.cfg.Sidebar.Indents.Guild)
	m.setNodeLineStyle(guildNode, m.guildNodeStyle(guild.ID))
	parent.AddChild(guildNode)
	m.nodes[discord.Snowflake(guild.ID)] = guildNode
}

func (m Model) createChannelNode(parent *tree.Node, channel discord.Channel) {
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
	m.nodes[discord.Snowflake(channel.ID)] = channelNode
}

// showVoiceMembers lists who is in each voice channel of the guild under it.
func (m *Model) showVoiceMembers(guildID discord.GuildID) {
	states, _ := m.state.Cabinet.VoiceStates(guildID)
	slices.SortFunc(states, func(a, b discord.VoiceState) int { return cmp.Compare(a.UserID, b.UserID) })
	channels, _ := m.state.Cabinet.Channels(guildID)
	for _, channel := range channels {
		node := m.nodes[discord.Snowflake(channel.ID)]
		if node == nil || !isVoice(channel.Type) {
			continue
		}
		// The members are replaced, so what is selected cannot stay one of them.
		if slices.Contains(node.Children(), m.selectionState.CurrentNode()) {
			m.selectionState.SetCurrentNode(node)
		}
		node.ClearChildren()
		for _, state := range states {
			if state.ChannelID == channel.ID {
				node.AddChild(tree.NewNode(m.voiceMemberText(guildID, state)).SetReference(state.UserID))
			}
		}
	}
	m.styleVoiceMembers()
}

// voiceMemberText returns the name of the member that state is of, and whether they are muted or deafened.
func (m Model) voiceMemberText(guildID discord.GuildID, state discord.VoiceState) string {
	text := state.UserID.String()
	member := state.Member
	if member == nil {
		member, _ = m.state.Cabinet.Member(guildID, state.UserID)
	}
	if member != nil {
		text = cmp.Or(member.Nick, member.User.DisplayName, member.User.Username)
	}
	switch {
	case state.Deaf || state.SelfDeaf:
		text += m.cfg.Icons.VoiceDeafened
	case state.Mute || state.SelfMute:
		text += m.cfg.Icons.VoiceMuted
	}
	return text
}

// styleVoiceMembers shows who is speaking in the voice channel joined.
func (m Model) styleVoiceMembers() {
	node := m.nodes[discord.Snowflake(m.voice.Status().ChannelID)]
	if node == nil {
		return
	}
	for _, member := range node.Children() {
		var style tview.Style
		if userID, _ := member.Reference().(discord.UserID); m.voice.Speaking(userID) {
			style = m.cfg.Theme.GuildsTree.SpeakingStyle.Style
		}
		m.setNodeLineStyle(member, style)
	}
}

// voiceFooter returns the state of the voice channel joined or being joined, or "" if there is none.
func (m Model) voiceFooter() string {
	status := m.voice.Status()
	channel, err := m.state.Cabinet.Channel(status.ChannelID)
	if err != nil {
		return ""
	}
	name := ui.ChannelToString(*channel, m.cfg.Icons, m.state)
	switch {
	case !status.Joined:
		return "Joining " + name + "..."
	case m.voice.Latency() == 0:
		return "In " + name
	default:
		return "In " + name + " (" + m.voice.Latency().Round(time.Millisecond).String() + ")"
	}
}

// voiceChannel returns the voice channel of node, or nil if it is not one.
func (m Model) voiceChannel(node *tree.Node) *discord.Channel {
	if node == nil {
		return nil
	}
	channelID, _ := node.Reference().(discord.ChannelID)
	channel, err := m.state.Cabinet.Channel(channelID)
	if err != nil || !isVoice(channel.Type) {
		return nil
	}
	return channel
}

// inVoice reports whether the voice channel is joined or being joined.
func (m Model) inVoice(channel discord.Channel) bool {
	return m.voice.Status().ChannelID == channel.ID
}

// toggleVoice joins the voice channel of node, or leaves it if it is joined.
func (m Model) toggleVoice(node *tree.Node) tview.Cmd {
	channel := m.voiceChannel(node)
	if channel == nil {
		return nil
	}
	leave := m.inVoice(*channel)
	return func() tview.Msg {
		if leave {
			if err := m.voice.Leave(); err != nil {
				return ui.ErrorDialog("leave "+channel.Name, err)
			}
		} else if err := m.voice.Join(channel.ID); err != nil {
			return ui.ErrorDialog("join "+channel.Name, err)
		}
		return nil
	}
}

// ToggleMute starts or stops sending what the user says to the voice channel joined.
func (m Model) ToggleMute() tview.Cmd {
	return toggleVoiceDevice("toggle mute", m.voice.ToggleMute)
}

// ToggleDeafen stops or starts playing what is said in the voice channel joined.
func (m Model) ToggleDeafen() tview.Cmd {
	return toggleVoiceDevice("toggle deafen", m.voice.ToggleDeafen)
}

func toggleVoiceDevice(what string, toggle func() error) tview.Cmd {
	return func() tview.Msg {
		if err := toggle(); err != nil {
			return ui.ErrorDialog(what, err)
		}
		return nil
	}
}

// expandable reports whether selecting node expands or collapses it, which the members of a voice channel do not make it.
func (m Model) expandable(node *tree.Node) bool {
	return len(node.Children()) > 0 && m.voiceChannel(node) == nil
}

func isVoice(t discord.ChannelType) bool {
	return t == discord.GuildVoice || t == discord.GuildStageVoice
}

func (m Model) setNodeLineStyle(node *tree.Node, style tview.Style) {
	line := node.Line()
	for i := range line {
		line[i].Style = style
	}
	node.SetLine(line)
}

func (m Model) createChannelNodes(node *tree.Node, channels []discord.Channel) {
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
			// Parent categories are inserted earlier in this function, so this lookup is O(1) and avoids per-channel subtree walks.
			parent := m.nodes[discord.Snowflake(channel.ParentID)]
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
	path := m.root.PathTo(node)
	if len(path) < 3 {
		return
	}
	parent := path[len(path)-2]
	parent.Collapse()
	m.selectionState.SetCurrentNode(parent)
}

// Update returns m changed in response to msg and a command to run, or nil.
func (m Model) Update(msg tview.Msg) (Model, tview.Cmd) {
	switch msg := msg.(type) {
	case *gateway.ReadyEvent:
		m.rebuild(msg)
		return m, nil
	case *gateway.MessageCreateEvent:
		if !msg.GuildID.IsValid() {
			m.moveDMToFront(msg.ChannelID)
		}
		return m, nil
	case *gateway.PresenceUpdateEvent:
		m.updateDMNodeStyle(msg.User.ID)
		return m, nil
	case *read.UpdateEvent:
		m.refreshReadStyles(msg)
		return m, nil
	case *gateway.VoiceStateUpdateEvent:
		m.showVoiceMembers(msg.GuildID)
		return m, nil
	case VoiceMsg:
		m.styleVoiceMembers()
		return m, nil

	case NavigateMsg:
		return m, m.navigate(msg.ChannelID)
	case Msg:
		m.selectionState.Apply(tree.Change(msg))
		return m, nil
	case SelectedMsg:
		m.selectionState.SetCurrentNode(msg.Node)
		return m, m.selectNode(msg.Node)
	case tview.KeyMsg:
		switch {
		case keybind.Matches(msg, m.cfg.Keybinds.GuildsTree.CollapseAll.Keybind):
			for _, node := range m.root.Children() {
				node.CollapseAll()
			}
			return m, nil
		case keybind.Matches(msg, m.cfg.Keybinds.GuildsTree.CollapseParentNode.Keybind):
			m.collapseParentNode(m.selectionState.CurrentNode())
			return m, nil
		case keybind.Matches(msg, m.cfg.Keybinds.GuildsTree.SelectPreviousUnread.Keybind, m.cfg.Keybinds.GuildsTree.SelectNextUnread.Keybind):
			previous := keybind.Matches(msg, m.cfg.Keybinds.GuildsTree.SelectPreviousUnread.Keybind)
			if node := adjacentNode(m.root, m.selectionState.CurrentNode(), m.isUnread, previous); node != nil {
				// A guild or the direct messages without channels yet: load them and move on to the unread one.
				if _, ok := node.Reference().(discord.ChannelID); !ok {
					m.selectNode(node)
					if channel := adjacentNode(node, node, m.isUnread, previous); channel != nil {
						node = channel
					}
				}
				m.expandPathToNode(node)
				m.selectionState.SetCurrentNode(node)
			}
			return m, nil
		case keybind.Matches(msg, m.cfg.Keybinds.GuildsTree.YankID.Keybind):
			return m, uitree.YankID(m.selectionState.CurrentNode())
		case keybind.Matches(msg, m.cfg.Keybinds.GuildsTree.ToggleVoice.Keybind):
			return m, m.toggleVoice(m.selectionState.CurrentNode())
		}
	}
	return m, nil
}

// isUnread reports whether node is an unread channel, or a guild or the direct messages whose channels are not shown yet and has one.
func (m Model) isUnread(node *tree.Node) bool {
	if len(node.Children()) > 0 {
		return false
	}
	opts := ningen.UnreadOpts{IncludeMutedCategories: true}
	switch ref := node.Reference().(type) {
	case discord.GuildID:
		return m.state.GuildIsUnread(ref, ningen.GuildUnreadOpts{UnreadOpts: opts}) != ningen.ChannelRead
	case discord.ChannelID:
		return m.state.ChannelIsUnread(ref, opts) != ningen.ChannelRead
	case dmNode:
		channels, _ := m.state.PrivateChannels()
		return slices.ContainsFunc(channels, func(c discord.Channel) bool { return m.state.ChannelIsUnread(c.ID, opts) != ningen.ChannelRead })
	}
	return false
}

// adjacentNode returns the nearest node under root matching match after current, or before it if previous, wrapping around, or nil if no other node matches.
func adjacentNode(root, current *tree.Node, match func(*tree.Node) bool, previous bool) *tree.Node {
	var nodes []*tree.Node
	after := 0
	root.Walk(func(node, _ *tree.Node) bool {
		if node == current {
			after = len(nodes)
		} else if match(node) {
			nodes = append(nodes, node)
		}
		return true
	})
	if len(nodes) == 0 {
		return nil
	}
	if previous {
		after += len(nodes) - 1
	}
	return nodes[after%len(nodes)]
}

func (m Model) findNodeByReference(reference any) *tree.Node {
	switch ref := reference.(type) {
	case discord.GuildID:
		return m.nodes[discord.Snowflake(ref)]
	case discord.ChannelID:
		return m.nodes[discord.Snowflake(ref)]
	case dmNode:
		return m.dmRootNode
	default:
		// Fallback keeps this helper safe for non-indexed custom references.
		var found *tree.Node
		m.root.Walk(func(node, _ *tree.Node) bool {
			if node.Reference() == reference {
				found = node
				return false
			}
			return true
		})
		return found
	}
}

func (m Model) findNodeByChannelID(channelID discord.ChannelID) *tree.Node {
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

func (m Model) expandPathToNode(node *tree.Node) {
	if node == nil {
		return
	}
	for _, n := range m.root.PathTo(node) {
		n.Expand()
	}
}

func unreadStyle(indication ningen.UnreadIndication) tview.Style {
	var style tview.Style
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
