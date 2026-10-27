package guildstree

import (
	"log/slog"

	"github.com/ayn2op/arikawa/v3/discord"
	"github.com/ayn2op/discordo/internal/ui"
	"github.com/ayn2op/ningen/v3"
	"github.com/ayn2op/tview"
	"github.com/ayn2op/tview/tree"
)

// currentMsg makes a node of the tree the current one.
type currentMsg struct {
	current *tree.Node
}

// selectedMsg is sent when the user selects a node.
type selectedMsg struct {
	node *tree.Node
}

type ChannelLoadedMsg struct {
	Channel  discord.Channel
	Messages []discord.Message
}

type NavigateMsg struct{ ChannelID discord.ChannelID }

// VoiceMsg is sent when who is speaking or the state of the voice channel joined changes.
type VoiceMsg struct{}

func (m *Model) navigate(channelID discord.ChannelID) tview.Cmd {
	channel, err := m.state.Cabinet.Channel(channelID)
	if err != nil {
		slog.Error("failed to get channel from state", "err", err, "channel_id", channelID)
		return nil
	}
	node := m.findNodeByChannelID(channelID)
	if node == nil {
		slog.Error("failed to locate channel in tree", "channel_id", channelID)
		return nil
	}
	m.expandPathToNode(node)
	m.setCurrentNode(node)
	if channel.Type == discord.GuildCategory {
		return nil
	}
	return m.selectNode(node)
}

func (m Model) selectNode(node *tree.Node) tview.Cmd {
	if m.expandable(node) {
		node.SetExpanded(!node.Expanded())
		return nil
	}

	switch ref := node.Reference().(type) {
	case discord.GuildID:
		m.expandGuild(node, ref)
	case discord.ChannelID:
		return m.selectChannel(node, ref)
	case dmNode:
		m.expandPrivateChannels(node)
	}

	return nil
}

func (m Model) expandGuild(node *tree.Node, guildID discord.GuildID) {
	go m.state.MemberState.Subscribe(guildID)
	channels, err := m.state.Cabinet.Channels(guildID)
	if err != nil {
		slog.Error("failed to get channels", "err", err, "guild_id", guildID)
		return
	}

	ui.SortGuildChannels(channels)
	m.createChannelNodes(node, channels)
	m.showVoiceMembers(guildID)
	node.Expand()
}

// selectChannel loads the messages of the channel, or shows the threads of a forum.
func (m Model) selectChannel(node *tree.Node, channelID discord.ChannelID) tview.Cmd {
	channel, err := m.state.Cabinet.Channel(channelID)
	if err != nil {
		slog.Error("failed to get channel from state", "err", err, "channel_id", channelID)
		return nil
	}

	if channel.Type != discord.GuildForum {
		return loadChannel(m.state, uint(m.cfg.MessagesLimit), *channel)
	}

	m.expandForum(node, *channel)
	return nil
}

func (m Model) expandForum(node *tree.Node, forum discord.Channel) {
	channels, err := m.state.Cabinet.Channels(forum.GuildID)
	if err != nil {
		slog.Error("failed to get channels for forum threads", "err", err, "guild_id", forum.GuildID)
		return
	}

	for _, child := range channels {
		if child.ParentID == forum.ID && isThread(child.Type) {
			m.createChannelNode(node, child)
		}
	}

	node.Expand()
}

func (m Model) expandPrivateChannels(node *tree.Node) {
	channels, err := m.state.PrivateChannels()
	if err != nil {
		slog.Error("failed to get private channels", "err", err)
		return
	}

	ui.SortPrivateChannels(channels)
	for _, channel := range channels {
		m.createChannelNode(node, channel)
	}

	node.Expand()
}

func loadChannel(state *ningen.State, limit uint, channel discord.Channel) tview.Cmd {
	return func() tview.Msg {
		messages, err := state.Messages(channel.ID, limit)
		if err != nil {
			slog.Error("failed to get messages", "err", err, "channel_id", channel.ID, "limit", limit)
			return nil
		}

		if lastMessageID := state.LastMessage(channel.ID); lastMessageID.IsValid() {
			go state.ReadState.MarkRead(channel.ID, lastMessageID)
		}

		return ChannelLoadedMsg{Channel: channel, Messages: messages}
	}
}
