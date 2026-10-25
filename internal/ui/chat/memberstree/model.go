package memberstree

import (
	"cmp"
	"fmt"
	"slices"
	"strings"

	"github.com/ayn2op/arikawa/v3/discord"
	"github.com/ayn2op/arikawa/v3/gateway"
	"github.com/ayn2op/arikawa/v3/state"
	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/discordo/internal/ui"
	uitree "github.com/ayn2op/discordo/internal/ui/tree"
	"github.com/ayn2op/ningen/v3"
	"github.com/ayn2op/tview"
	"github.com/ayn2op/tview/keybind"
	"github.com/ayn2op/tview/text"
	"github.com/ayn2op/tview/tree"
	"github.com/ayn2op/tview/viewport"
	"github.com/gdamore/tcell/v3"
)

// groupRef is the ID of a member list group: a role ID, "online", or "offline".
type groupRef string

func (r groupRef) String() string {
	return string(r)
}

// memberRef is a member and its index in the guild member list, which decides the chunk of the list to request.
type memberRef struct {
	userID discord.UserID
	index  int
}

func (r memberRef) String() string {
	return r.userID.String()
}

// Model is a tree of the members of the selected channel, grouped as Discord groups them.
type Model struct {
	root        *tree.Node
	current     *tree.Node
	scrollState viewport.ScrollState

	cfg   *config.Config
	state *ningen.State

	channel *discord.Channel
	// nodes indexes the tree by group and user ID, so that a rebuild reuses the nodes and keeps the selection and collapsed groups.
	nodes map[string]*tree.Node

	// focused reports whether the tree takes keys.
	focused bool
}

func NewModel(cfg *config.Config, state *ningen.State) Model {
	return Model{
		root:  tree.NewNode(""),
		cfg:   cfg,
		state: state,
		nodes: make(map[string]*tree.Node),
	}
}

// Shown reports whether the selected channel has members to show: guild channels other than threads, and group DMs.
func (m Model) Shown() bool {
	channel := m.channel
	if channel == nil {
		return false
	}
	switch channel.Type {
	case discord.GroupDM:
		return true
	case discord.GuildPublicThread, discord.GuildPrivateThread, discord.GuildAnnouncementThread:
		return false
	default:
		return channel.GuildID.IsValid()
	}
}

// SetChannel shows the members of channel and requests its member list.
func (m *Model) SetChannel(channel *discord.Channel) tview.Cmd {
	m.channel = channel
	m.rebuild()
	return m.requestMembers(0)
}

// SetFocused sets whether the tree takes keys.
func (m *Model) SetFocused(focused bool) {
	m.focused = focused
}

// setCurrentNode selects node and scrolls to it.
func (m *Model) setCurrentNode(node *tree.Node) {
	m.current = node
	m.scrollState.ScrollToTarget()
}

// rebuild recreates the tree from the member list of the selected channel.
func (m Model) rebuild() {
	m.root.ClearChildren()
	channel := m.channel
	switch {
	case !m.Shown():
	case channel.Type == discord.GroupDM:
		m.addRecipients(channel.DMRecipients)
	default:
		m.addGuildMembers(channel)
	}

	clear(m.nodes)
	m.root.Walk(func(node, _ *tree.Node) bool {
		if ref, ok := node.Reference().(fmt.Stringer); ok {
			m.nodes[ref.String()] = node
		}

		return true
	})
}

func (m Model) addGuildMembers(channel *discord.Channel) {
	list, err := m.state.MemberState.GetMemberList(channel.GuildID, channel.ID)
	if err != nil {
		// The list has yet to arrive.
		return
	}

	list.ViewItems(func(items []gateway.GuildMemberListOpItem) {
		m.addListItems(channel.GuildID, items)
	})
}

func (m Model) addListItems(guildID discord.GuildID, items []gateway.GuildMemberListOpItem) {
	var group *tree.Node
	for index, item := range items {
		if item.Group != nil {
			group = m.addGroup(guildID, *item.Group)
			continue
		}

		if item.Member == nil {
			group = nil
			continue
		}

		// Members after an unloaded range belong to a group that has yet to arrive.
		if group == nil {
			group = m.addGroup(guildID, gateway.GuildMemberListGroup{})
		}

		m.addMember(group, guildID, item.Member.Member, index)
	}
}

func (m Model) addRecipients(recipients []discord.User) {
	recipients = slices.Clone(recipients)
	// Discord leaves the current user out of the recipients.
	if me, err := m.state.Cabinet.Me(); err == nil {
		recipients = append(recipients, *me)
	}
	rank := func(user discord.User) int { return statusRank(m.status(discord.NullGuildID, user.ID)) }
	name := func(user discord.User) string { return strings.ToLower(user.DisplayOrUsername()) }
	// By status, online first, then by name.
	slices.SortFunc(recipients, func(a, b discord.User) int {
		return cmp.Or(cmp.Compare(rank(a), rank(b)), strings.Compare(name(a), name(b)))
	})

	group := m.addGroup(discord.NullGuildID, gateway.GuildMemberListGroup{ID: "members", Count: uint64(len(recipients))})
	for _, recipient := range recipients {
		m.addMember(group, discord.NullGuildID, discord.Member{User: recipient}, -1)
	}
}

func (m Model) addGroup(guildID discord.GuildID, group gateway.GuildMemberListGroup) *tree.Node {
	name, style := m.groupName(guildID, group.ID)
	if group.Count > 0 {
		name = fmt.Sprintf("%s — %d", name, group.Count)
	}

	ref := groupRef(group.ID)
	node := m.reuse(ref.String()).
		SetSpans(text.Span{Text: name, Style: style}).
		SetReference(ref).
		SetExpandable(true)
	m.root.AddChild(node)
	return node
}

// groupName returns the name of the group with groupID, in the color of its role if it is one.
func (m Model) groupName(guildID discord.GuildID, groupID string) (string, tview.Style) {
	switch groupID {
	case "online":
		return "Online", tview.Style{}
	case "offline":
		return "Offline", tview.Style{}
	}

	roleID, err := discord.ParseSnowflake(groupID)
	if err != nil {
		return defaultGroupName, tview.Style{}
	}

	role, err := m.state.Cabinet.Role(guildID, discord.RoleID(roleID))
	if err != nil {
		return defaultGroupName, tview.Style{}
	}

	var style tview.Style
	if role.Color != 0 {
		style = style.Foreground(tcell.NewHexColor(int32(role.Color)))
	}

	return role.Name, style
}

// isGroup reports whether node is a member list group.
func isGroup(node *tree.Node) bool {
	if node == nil {
		return false
	}

	_, ok := node.Reference().(groupRef)
	return ok
}

// addMember adds a member as Discord shows it: a status dot, then the name in the color of their highest colored role, dimmed when offline.
func (m Model) addMember(group *tree.Node, guildID discord.GuildID, member discord.Member, index int) {
	name := cmp.Or(member.Nick, member.User.DisplayName, member.User.Username)
	var nameStyle tview.Style
	color, ok := state.MemberColor(&member, func(id discord.RoleID) *discord.Role {
		role, _ := m.state.Cabinet.Role(guildID, id)
		return role
	})
	if ok {
		nameStyle = nameStyle.Foreground(tcell.NewHexColor(int32(color)))
	}
	spans := []text.Span{{Text: statusMarker}, {Text: " "}, {Text: name, Style: nameStyle}}
	m.setStatus(spans, m.status(guildID, member.User.ID))
	group.AddChild(m.reuse(member.User.ID.String()).SetSpans(spans...).SetReference(memberRef{userID: member.User.ID, index: index}))
}

const (
	statusMarker = "●"
	// defaultGroupName is the name of a group that is not a known role.
	defaultGroupName = "Members"
)

// setStatus styles the status dot and name of a member's spans for status.
func (m Model) setStatus(spans []text.Span, status discord.Status) {
	spans[0].Style = m.cfg.Theme.MembersTree.StatusStyle(status)
	spans[2].Style = spans[2].Style.Dim(status == discord.OfflineStatus)
}

// reuse takes the node of id out of the index, so that it is reused once, or else returns a new node. Either way the node has no children.
func (m Model) reuse(id string) *tree.Node {
	node, ok := m.nodes[id]
	if !ok {
		return tree.NewNode("")
	}
	delete(m.nodes, id)
	return node.ClearChildren()
}

func (m Model) refreshStatus(userID discord.UserID) {
	if node, ok := m.nodes[userID.String()]; ok {
		spans := node.Spans()
		m.setStatus(spans, m.status(m.channel.GuildID, userID))
		node.SetSpans(spans...)
	}
}

func (m Model) status(guildID discord.GuildID, userID discord.UserID) discord.Status {
	presence, err := m.state.Cabinet.Presence(guildID, userID)
	if err != nil {
		return discord.OfflineStatus
	}
	return presence.Status
}

// statusRank orders members as Discord does: online, idle, do not disturb, then offline.
func statusRank(status discord.Status) int {
	switch status {
	case discord.OnlineStatus:
		return 0
	case discord.IdleStatus:
		return 1
	case discord.DoNotDisturbStatus:
		return 2
	default:
		return 3
	}
}

func (m Model) parent(node *tree.Node) *tree.Node {
	path := m.root.PathTo(node)
	if len(path) < 3 {
		return nil
	}
	return path[len(path)-2]
}

var _ tview.Model[Model] = Model{}

func (m Model) Init() tview.Cmd { return nil }

// Update returns m changed in response to msg and a command to run, or nil.
func (m Model) Update(msg tview.Msg) (Model, tview.Cmd) {
	switch msg := msg.(type) {
	case *gateway.ReadyEvent:
		// ningen drops its member lists on READY, so request the list again.
		m.rebuild()
		return m, m.requestMembers(0)
	case *gateway.GuildMemberListUpdateEvent:
		if channel := m.channel; channel != nil && channel.GuildID == msg.GuildID && m.state.MemberState.ListID(channel) == msg.ID {
			m.rebuild()
		}
		return m, nil
	case *gateway.PresenceUpdateEvent:
		m.refreshStatus(msg.User.ID)
		return m, nil

	case currentMsg:
		m.current = msg.current
		m.scrollState.ScrollToTarget()
		return m, m.requestSelectedMembers()
	case uitree.ScrollMsg:
		m.scrollState = msg.ScrollState
		if msg.Msg != nil {
			return m.Update(msg.Msg)
		}
		return m, nil
	case selectedMsg:
		m.setCurrentNode(msg.node)
		if isGroup(msg.node) {
			msg.node.SetExpanded(!msg.node.Expanded())
		}
		return m, m.requestSelectedMembers()
	case tview.KeyMsg:
		keybinds := m.cfg.Keybinds.MembersTree
		switch {
		case keybind.Matches(msg, keybinds.CollapseAll.Keybind):
			for _, node := range m.root.Children() {
				node.Collapse()
			}

			return m, nil
		case keybind.Matches(msg, keybinds.CollapseParentNode.Keybind):
			if parent := m.parent(m.current); parent != nil {
				parent.Collapse()
				m.setCurrentNode(parent)
			}

			return m, nil
		case keybind.Matches(msg, keybinds.YankID.Keybind):
			return m, uitree.YankID(m.current)
		}
	}

	return m, nil
}

// View shows the tree in a box titled Members.
func (m Model) View() tview.Widget {
	onChange := func(current *tree.Node) tview.Msg { return currentMsg{current} }
	onSelect := func(n *tree.Node) tview.Msg { return selectedMsg{node: n} }
	return uitree.New(
		m.root,
		m.current,
		m.scrollState,
		m.cfg,
		m.cfg.Theme.MembersTree,
		m.cfg.UI.MembersTree.Graphics,
		m.cfg.Keybinds.MembersTree,
		m.focused,
		onChange,
		onSelect,
	).
		Title(ui.Label("Members", m.cfg.Theme.Title, m.focused))
}
