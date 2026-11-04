package guildstree

import (
	"strconv"

	"github.com/ayn2op/arikawa/v3/discord"
	"github.com/ayn2op/discordo/internal/favorites"
	"github.com/ayn2op/tview/tree"
)

const favoritesNodeName = "Favorites"

// favoritesRoot is the reference of the node the bookmarked channels sit under.
type favoritesRoot struct{}

// favoriteNode is the reference of a bookmarked channel's node. It is not a discord.ChannelID so that
// the channel's own node, the one under its guild or the direct messages, stays the one indexed by id.
type favoriteNode struct {
	slot      int
	channelID discord.ChannelID
}

// SetFavorites shows the channels store bookmarks, doing nothing while they are the ones already shown.
func (m *Model) SetFavorites(store favorites.Store) {
	if m.favorites.Equal(store) {
		return
	}

	m.favorites = store
	m.fillFavorites()
}

// fillFavorites puts a node under the favorites section for each bookmarked channel, in slot order,
// and takes the section out of the tree while nothing is bookmarked.
func (m *Model) fillFavorites() {
	clear(m.favoriteNodes)
	m.favoritesRootNode.ClearChildren()

	for slot, favorite := range m.favorites.Channels() {
		m.createFavoriteNode(slot, favorite.ChannelID)
	}

	setNodeText(m.favoritesRootNode, m.favoritesNodeText())
	m.showFavorites(len(m.favoritesRootNode.Children()) > 0)
}

// showFavorites puts the favorites section first among the tree's sections, or takes it out.
func (m *Model) showFavorites(shown bool) {
	children := make([]*tree.Node, 0, len(m.root.Children())+1)
	if shown {
		children = append(children, m.favoritesRootNode)
	}

	for _, child := range m.root.Children() {
		if child != m.favoritesRootNode {
			children = append(children, child)
		}
	}

	m.root.SetChildren(children)
}

// createFavoriteNode adds the node that opens channelID from slot, if the channel is one that opens messages.
func (m *Model) createFavoriteNode(slot int, channelID discord.ChannelID) {
	if !channelID.IsValid() {
		return
	}

	channel, err := m.state.Cabinet.Channel(channelID)
	// A channel left behind, or one hand-written into the favorites file, is bookmarked but not shown.
	if err != nil {
		return
	}

	// Only channels that open messages are listed: the slots hold no other kind unless the file was written by hand.
	switch channel.Type {
	case discord.GuildCategory, discord.GuildForum:
		return
	}

	node := tree.NewNode(m.favoriteText(slot, *channel)).
		SetReference(favoriteNode{slot: slot, channelID: channelID}).
		SetIndent(m.cfg.Sidebar.Indents.Channel)
	setNodeNameStyle(node, m.channelNodeStyle(*channel))
	m.favoritesRootNode.AddChild(node)
	m.favoriteNodes[channelID] = node
}

// favoritesNodeText returns the name of the favorites section with the unread mentions of its channels after it.
func (m Model) favoritesNodeText() string {
	channels := make([]discord.Channel, 0, len(m.favorites.Channels()))
	for _, favorite := range m.favorites.Channels() {
		if channel, err := m.state.Cabinet.Channel(favorite.ChannelID); err == nil {
			channels = append(channels, *channel)
		}
	}

	return favoritesNodeName + mentionsLabel(m.mentions(channels...))
}

// favoriteText returns the name of a bookmarked channel with the key that opens it before it, as the
// same channel name is read in several guilds and the slot is what tells the rows apart.
func (m Model) favoriteText(slot int, channel discord.Channel) string {
	return m.favoriteLabel(slot) + " " + m.channelText(channel)
}

// favoriteLabel returns the key bound to slot, falling back to its number where it is bound to none.
func (m Model) favoriteLabel(slot int) string {
	if keybinds := m.cfg.Keybinds.Favorites; slot < len(keybinds) {
		if key := keybinds[slot].Help().Key; key != "" {
			return key
		}
	}

	return strconv.Itoa(slot + 1)
}

// nodesShowing returns every node that shows channelID: the one under its guild or the direct messages, and its bookmark.
func (m Model) nodesShowing(channelID discord.ChannelID) []*tree.Node {
	nodes := make([]*tree.Node, 0, 2)
	for _, node := range []*tree.Node{m.channelNodes[channelID], m.favoriteNodes[channelID]} {
		if node != nil {
			nodes = append(nodes, node)
		}
	}

	return nodes
}

// CurrentChannelID returns the channel the highlighted node opens, and whether it opens one.
func (m Model) CurrentChannelID() (discord.ChannelID, bool) {
	if m.current == nil {
		return discord.NullChannelID, false
	}

	switch ref := m.current.Reference().(type) {
	case discord.ChannelID:
		return ref, true
	case favoriteNode:
		return ref.channelID, true
	}

	return discord.NullChannelID, false
}
