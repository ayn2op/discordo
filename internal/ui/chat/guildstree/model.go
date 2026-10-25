package guildstree

import (
	"cmp"
	"errors"
	"fmt"
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
	"github.com/ayn2op/tview/text"
	"github.com/ayn2op/tview/tree"
	"github.com/ayn2op/tview/viewport"
	"github.com/gdamore/tcell/v3"
)

type dmNode struct{}

const (
	dmNodeName        = "Direct Messages"
	maxMentions       = 99
	mentionsFormat    = " (%d)"
	maxMentionsFormat = " (%d+)"
)

type Model struct {
	root        *tree.Node
	current     *tree.Node
	scrollState viewport.ScrollState

	cfg          *config.Config
	state        *ningen.State
	voiceSession *voice.Session

	// nodes indexes the guild and channel nodes for frequent event handlers (read updates, picker navigation).
	// It mirrors the current rendered tree and is rebuilt on READY before nodes are added.
	nodes      map[discord.Snowflake]*tree.Node
	dmRootNode *tree.Node

	// focused reports whether the tree takes keys.
	focused bool
}

func NewModel(cfg *config.Config, state *ningen.State) Model {
	return Model{
		root:         tree.NewNode(""),
		cfg:          cfg,
		state:        state,
		voiceSession: voice.New(state.State, cfg.Voice.Sensitivity),
		nodes:        make(map[discord.Snowflake]*tree.Node),
	}
}

// CurrentNode returns the selected node, or nil for none.
func (m Model) CurrentNode() *tree.Node {
	return m.current
}

// SetFocused sets whether the tree takes keys.
func (m *Model) SetFocused(focused bool) {
	m.focused = focused
}

func (m *Model) reset() *tree.Node {
	// Keep allocated map capacity; READY can rebuild often during reconnects.
	clear(m.nodes)
	m.dmRootNode = tree.NewNode(m.dmNodeText()).
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
	for index := range event.Guilds {
		guildsByID[event.Guilds[index].ID] = &event.Guilds[index]
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
		// A guild in a folder of its own is shown as if it were in none.
		if folder.ID != 0 || len(folder.GuildIDs) != 1 {
			m.createFolderNode(folder, guildsByID)
			continue
		}

		if guild := guildsByID[folder.GuildIDs[0]]; guild != nil {
			m.createGuildNode(root, guild.Guild)
		}
	}

	m.setCurrentNode(root)
}

func (m Model) refreshReadState(event *read.UpdateEvent) {
	if event.GuildID.IsValid() {
		m.refreshGuildNode(event.GuildID)
	} else if m.dmRootNode != nil {
		setNodeText(m.dmRootNode, m.dmNodeText())
	}

	node := m.findNodeByRef(event.ChannelID)
	if node == nil {
		return
	}
	channel, err := m.state.Cabinet.Channel(event.ChannelID)
	if err != nil {
		indication := m.state.ChannelIsUnread(event.ChannelID, ningen.UnreadOpts{IncludeMutedCategories: true})
		setNodeNameStyle(node, unreadStyle(indication))
		return
	}

	setNodeText(node, m.channelText(*channel))
	setNodeNameStyle(node, m.channelNodeStyle(*channel))
}

func (m Model) refreshGuildNode(guildID discord.GuildID) {
	node := m.findNodeByRef(guildID)
	if node == nil {
		return
	}

	if guild, err := m.state.Cabinet.Guild(guildID); err == nil {
		setNodeText(node, m.guildText(*guild))
	}

	setNodeNameStyle(node, m.guildNodeStyle(guildID))
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
	setNodeNameStyle(node, m.channelNodeStyle(*channel))
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
		setNodeNameStyle(folderNode, folderStyle)
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
	guildNode := tree.NewNode(m.guildText(guild)).
		SetReference(guild.ID).
		SetExpandable(true).
		SetExpanded(false).
		SetIndent(m.cfg.Sidebar.Indents.Guild)
	setNodeNameStyle(guildNode, m.guildNodeStyle(guild.ID))
	parent.AddChild(guildNode)
	m.nodes[discord.Snowflake(guild.ID)] = guildNode
}

func (m Model) createChannelNode(parent *tree.Node, channel discord.Channel) {
	needsViewPermission := !slices.Contains([]discord.ChannelType{discord.DirectMessage, discord.GroupDM, discord.GuildCategory}, channel.Type)
	if needsViewPermission && !m.state.HasPermissions(channel.ID, discord.PermissionViewChannel) {
		return
	}

	indents := m.cfg.Sidebar.Indents
	channelNode := tree.NewNode(m.channelText(channel)).SetReference(channel.ID)
	setNodeNameStyle(channelNode, m.channelNodeStyle(channel))
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
	states, _ := m.state.VoiceStates(guildID)
	slices.SortFunc(states, func(a, b discord.VoiceState) int { return cmp.Compare(a.UserID, b.UserID) })

	channels, _ := m.state.Cabinet.Channels(guildID)
	for _, channel := range channels {
		node := m.nodes[discord.Snowflake(channel.ID)]
		if node == nil || !isVoice(channel.Type) {
			continue
		}

		m.setVoiceMembers(node, channel, states)
	}

	m.styleVoiceMembers()
}

// setVoiceMembers lists under node, of the voice channel, those of states who are in it.
func (m *Model) setVoiceMembers(node *tree.Node, channel discord.Channel, states []discord.VoiceState) {
	// The members are replaced, so what is selected cannot stay one of them.
	if slices.Contains(node.Children(), m.current) {
		m.current = node
	}

	node.ClearChildren()
	for _, state := range states {
		if state.ChannelID == channel.ID {
			node.AddChild(tree.NewNode(m.voiceMemberText(channel.GuildID, state)).SetReference(state.UserID))
		}
	}

	// After its name, how full it is, if it can be.
	spans := node.Spans()[:1]
	if channel.VoiceUserLimit > 0 {
		label := fmt.Sprintf("%d/%d", len(node.Children()), channel.VoiceUserLimit)
		spans = append(spans, text.Span{Text: " "}, text.Span{Text: label, Style: m.cfg.Theme.GuildsTree.VoiceLimitStyle.Style})
	}

	node.SetSpans(spans...)
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

	return text + m.voiceMarkers(state)
}

// voiceMarkers returns what shows that the user that state is of is muted or deafened.
func (m Model) voiceMarkers(state discord.VoiceState) (markers string) {
	if state.Mute || state.SelfMute {
		markers += m.cfg.Icons.VoiceMuted
	}

	if state.Deaf || state.SelfDeaf {
		markers += m.cfg.Icons.VoiceDeafened
	}

	return markers
}

// styleVoiceMembers shows who is speaking in the voice channel joined.
func (m Model) styleVoiceMembers() {
	node := m.nodes[discord.Snowflake(m.voiceSession.Status().ChannelID)]
	if node == nil {
		return
	}

	for _, member := range node.Children() {
		var style tview.Style
		if userID, _ := member.Reference().(discord.UserID); m.voiceSession.Speaking(userID) {
			style = m.cfg.Theme.GuildsTree.SpeakingStyle.Style
		}

		setNodeNameStyle(member, style)
	}
}

// voiceFooter returns the state of the voice channel joined or being joined and of the user in it, or "" if there is none.
func (m Model) voiceFooter() string {
	status := m.voiceSession.Status()
	channel, err := m.state.Cabinet.Channel(status.ChannelID)
	if err != nil {
		return ""
	}

	name := ui.ChannelToString(*channel, m.cfg.Icons, m.state)
	if me, _ := m.state.Cabinet.Me(); me != nil {
		if state, _ := m.state.VoiceState(channel.GuildID, me.ID); state != nil {
			name += m.voiceMarkers(*state)
		}
	}

	switch {
	case !status.Joined:
		return "Joining " + name + "..."
	case m.voiceSession.Latency() == 0:
		return "In " + name
	default:
		return "In " + name + " (" + m.voiceSession.Latency().Round(time.Millisecond).String() + ")"
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

// joinable returns why the voice channel of node cannot be joined, or nil if it can be.
func (m Model) joinable(node *tree.Node, channel discord.Channel) error {
	// The users in the channel are the children of its node.
	atUserLimit := channel.VoiceUserLimit > 0 && len(node.Children()) >= int(channel.VoiceUserLimit)

	switch {
	case !m.state.HasPermissions(channel.ID, discord.PermissionConnect):
		return errors.New("you are not permitted to connect")
	// Discord does not answer a request to join a full voice channel, unless it is from someone who can move members.
	case atUserLimit && !m.state.HasPermissions(channel.ID, discord.PermissionMoveMembers):
		return errors.New("it is full")
	}

	return nil
}

// toggleVoice joins the voice channel of node, or leaves it if it is joined.
func (m Model) toggleVoice(node *tree.Node) tview.Cmd {
	channel := m.voiceChannel(node)
	switch {
	case channel == nil:
		return nil
	case m.voiceSession.Status().ChannelID == channel.ID:
		return voiceCmd("leave "+channel.Name, m.voiceSession.Leave)
	}

	err := m.joinable(node, *channel)
	return voiceCmd("join "+channel.Name, func() error {
		if err != nil {
			return err
		}

		return m.voiceSession.Join(channel.ID)
	})
}

// toggleMute starts or stops sending what the user says to the voice channel joined, if they are permitted to speak in it without holding a key.
func (m Model) toggleMute() tview.Cmd {
	return voiceCmd("toggle mute", func() error {
		if channelID := m.voiceSession.Status().ChannelID; channelID.IsValid() && !m.state.HasPermissions(channelID, discord.PermissionSpeak|discord.PermissionUseVAD) {
			return errors.New("you are not permitted to speak by voice activity")
		}

		return m.voiceSession.Toggle(false)
	})
}

// voiceCmd returns a command that does action, and shows why it failed to do what if it did.
func voiceCmd(what string, action func() error) tview.Cmd {
	return func() tview.Msg {
		if err := action(); err != nil {
			return ui.ErrorDialog(what, err)
		}

		return nil
	}
}

// expandable reports whether selecting node expands or collapses it, which the members of a voice channel do not make it.
func (m Model) expandable(node *tree.Node) bool {
	return len(node.Children()) > 0 && m.voiceChannel(node) == nil
}

func isVoice(channelType discord.ChannelType) bool {
	return channelType == discord.GuildVoice || channelType == discord.GuildStageVoice
}

// setNodeNameStyle styles the name of node, which is the first span of its text.
func setNodeNameStyle(node *tree.Node, style tview.Style) {
	spans := node.Spans()
	spans[0].Style = style
	node.SetSpans(spans...)
}

// setNodeText sets the text of the name of node, keeping its style.
func setNodeText(node *tree.Node, name string) {
	spans := node.Spans()
	spans[0].Text = name
	node.SetSpans(spans...)
}

// channelText returns the name of channel with its unread mentions after it.
func (m Model) channelText(channel discord.Channel) string {
	return ui.ChannelToString(channel, m.cfg.Icons, m.state) + mentionsLabel(m.mentions(channel))
}

// guildText returns the name of guild with the unread mentions in its channels after it.
func (m Model) guildText(guild discord.Guild) string {
	channels, _ := m.state.Cabinet.Channels(guild.ID)
	return guild.Name + mentionsLabel(m.mentions(channels...))
}

// dmNodeText returns the name of the direct messages node with their unread mentions after it.
func (m Model) dmNodeText() string {
	channels, _ := m.state.Cabinet.PrivateChannels()
	return dmNodeName + mentionsLabel(m.mentions(channels...))
}

// mentions returns the number of unread mentions in channels.
func (m Model) mentions(channels ...discord.Channel) (total int) {
	for _, channel := range channels {
		if state := m.state.ReadState.ReadState(channel.ID); state != nil {
			total += state.MentionCount
		}
	}
	return total
}

// mentionsLabel returns what follows a name with count unread mentions, or nothing if there are none.
func mentionsLabel(count int) string {
	switch {
	case count == 0:
		return ""
	case count > maxMentions:
		return fmt.Sprintf(maxMentionsFormat, maxMentions)
	}
	return fmt.Sprintf(mentionsFormat, count)
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
		if _, hasChild := hasChildByParentID[channel.ID]; hasChild && channel.Type == discord.GuildCategory {
			m.createChannelNode(node, channel)
		}
	}

	for _, channel := range channels {
		if !channel.ParentID.IsValid() {
			continue
		}

		// Parent categories are inserted earlier in this function, so this lookup is O(1) and avoids per-channel subtree walks.
		if parent := m.nodes[discord.Snowflake(channel.ParentID)]; parent != nil {
			m.createChannelNode(parent, channel)
		}
	}
}

func isThread(channelType discord.ChannelType) bool {
	switch channelType {
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
	m.setCurrentNode(parent)
}

// setCurrentNode selects node and scrolls to it.
func (m *Model) setCurrentNode(node *tree.Node) {
	m.current = node
	m.scrollState.ScrollToTarget()
}

// selectAdjacentUnread selects the next unread node, or the previous one if previous.
func (m *Model) selectAdjacentUnread(previous bool) {
	node := adjacentNode(m.root, m.current, m.isUnread, previous)
	if node == nil {
		return
	}

	// A guild or the direct messages without channels yet: load them and move on to the unread one.
	if _, ok := node.Reference().(discord.ChannelID); !ok {
		m.selectNode(node)
		if channel := adjacentNode(node, node, m.isUnread, previous); channel != nil {
			node = channel
		}
	}

	m.expandPathToNode(node)
	m.setCurrentNode(node)
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
		return slices.ContainsFunc(channels, func(channel discord.Channel) bool {
			return m.state.ChannelIsUnread(channel.ID, opts) != ningen.ChannelRead
		})
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

func (m Model) findNodeByRef(ref any) *tree.Node {
	switch ref := ref.(type) {
	case discord.GuildID:
		return m.nodes[discord.Snowflake(ref)]
	case discord.ChannelID:
		return m.nodes[discord.Snowflake(ref)]
	case dmNode:
		return m.dmRootNode
	}

	// Fallback keeps this helper safe for non-indexed custom references.
	var found *tree.Node
	m.root.Walk(func(node, _ *tree.Node) bool {
		if node.Reference() == ref {
			found = node
		}

		return found == nil
	})

	return found
}

func (m Model) findNodeByChannelID(channelID discord.ChannelID) *tree.Node {
	channel, err := m.state.Cabinet.Channel(channelID)
	if err != nil {
		slog.Error("failed to get channel", "channel_id", channelID, "err", err)
		return nil
	}

	var ref any
	if guildID := channel.GuildID; guildID.IsValid() {
		ref = guildID
	} else {
		ref = dmNode{}
	}
	if parent := m.findNodeByRef(ref); parent != nil && len(parent.Children()) == 0 {
		m.selectNode(parent)
	}

	return m.findNodeByRef(channelID)
}

func (m Model) expandPathToNode(node *tree.Node) {
	if node == nil {
		return
	}
	for _, ancestor := range m.root.PathTo(node) {
		ancestor.Expand()
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

var _ tview.Model[Model] = Model{}

// Init waits for who is speaking or the state of the voice channel joined to change.
func (m Model) Init() tview.Cmd {
	return func() tview.Msg {
		<-m.voiceSession.Updates()
		return VoiceMsg{}
	}
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
		m.refreshReadState(msg)
		return m, nil
	case *gateway.VoiceStateUpdateEvent:
		m.showVoiceMembers(msg.GuildID)
		return m, nil
	case VoiceMsg:
		m.styleVoiceMembers()
		return m, m.Init()

	case NavigateMsg:
		return m, m.navigate(msg.ChannelID)
	case currentMsg:
		m.current = msg.current
		m.scrollState.ScrollToTarget()
		return m, nil
	case uitree.ScrollMsg:
		m.scrollState = msg.ScrollState
		if msg.Msg != nil {
			return m.Update(msg.Msg)
		}
		return m, nil
	case selectedMsg:
		m.setCurrentNode(msg.node)
		return m, m.selectNode(msg.node)
	case tview.KeyMsg:
		keybinds := m.cfg.Keybinds.GuildsTree
		switch {
		case keybind.Matches(msg, keybinds.CollapseAll.Keybind):
			for _, node := range m.root.Children() {
				node.CollapseAll()
			}

			return m, nil
		case keybind.Matches(msg, keybinds.CollapseParentNode.Keybind):
			m.collapseParentNode(m.current)
			return m, nil
		case keybind.Matches(msg, keybinds.SelectPreviousUnread.Keybind, keybinds.SelectNextUnread.Keybind):
			m.selectAdjacentUnread(keybind.Matches(msg, keybinds.SelectPreviousUnread.Keybind))
			return m, nil
		case keybind.Matches(msg, keybinds.YankID.Keybind):
			return m, uitree.YankID(m.current)
		case keybind.Matches(msg, keybinds.ToggleVoice.Keybind):
			return m, m.toggleVoice(m.current)
		case keybind.Matches(msg, m.cfg.Keybinds.ToggleMute.Keybind):
			return m, m.toggleMute()
		case keybind.Matches(msg, m.cfg.Keybinds.ToggleDeafen.Keybind):
			return m, voiceCmd("toggle deafen", func() error { return m.voiceSession.Toggle(true) })
		case keybind.Matches(msg, m.cfg.Keybinds.LeaveVoice.Keybind):
			return m, voiceCmd("leave voice", m.voiceSession.Leave)
		}
	}

	return m, nil
}

// View shows the tree in a box titled Guilds.
func (m Model) View() tview.Widget {
	onChange := func(current *tree.Node) tview.Msg { return currentMsg{current} }
	onSelect := func(n *tree.Node) tview.Msg { return selectedMsg{node: n} }
	return uitree.New(
		m.root,
		m.current,
		m.scrollState,
		m.cfg,
		m.cfg.Theme.GuildsTree.TreeThemeConfig,
		m.cfg.UI.GuildsTree.Graphics,
		m.cfg.Keybinds.GuildsTree.TreeKeybinds,
		m.focused,
		onChange,
		onSelect,
	).
		Title(ui.Label("Guilds", m.cfg.Theme.Title, m.focused)).
		Footer(ui.Label(m.voiceFooter(), m.cfg.Theme.Footer, m.focused))
}
