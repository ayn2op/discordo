package guildstree

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/ayn2op/arikawa/v3/discord"
	"github.com/ayn2op/discordo/internal/ui"
	"github.com/ayn2op/ningen/v3"
	"github.com/ayn2op/tview"
	"github.com/ayn2op/tview/tree"
	"golang.design/x/clipboard"
)

type ChannelLoadedMsg struct {
	Channel  discord.Channel
	Messages []discord.Message
}

type NavigateMsg struct{ ChannelID discord.ChannelID }

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
	m.SetCurrentNode(node)
	if channel.Type == discord.GuildCategory {
		return nil
	}
	return m.selectNode(node)
}

func (m *Model) selectNode(node *tree.Node) tview.Cmd {
	if len(node.Children()) != 0 {
		node.SetExpanded(!node.Expanded())
		return nil
	}

	switch ref := node.Reference().(type) {
	case discord.GuildID:
		go m.state.MemberState.Subscribe(ref)
		channels, err := m.state.Cabinet.Channels(ref)
		if err != nil {
			slog.Error("failed to get channels", "err", err, "guild_id", ref)
			return nil
		}
		ui.SortGuildChannels(channels)
		m.createChannelNodes(node, channels)
		node.Expand()
	case discord.ChannelID:
		channel, err := m.state.Cabinet.Channel(ref)
		if err != nil {
			slog.Error("failed to get channel from state", "err", err, "channel_id", ref)
			return nil
		}
		if channel.Type != discord.GuildForum {
			return loadChannel(m.state, uint(m.cfg.MessagesLimit), *channel)
		}
		channels, err := m.state.Cabinet.Channels(channel.GuildID)
		if err != nil {
			slog.Error("failed to get channels for forum threads", "err", err, "guild_id", channel.GuildID)
			return nil
		}
		for _, child := range channels {
			if child.ParentID == channel.ID && isThread(child.Type) {
				m.createChannelNode(node, child)
			}
		}
		node.Expand()
	case dmNode:
		channels, err := m.state.PrivateChannels()
		if err != nil {
			slog.Error("failed to get private channels", "err", err)
			return nil
		}
		ui.SortPrivateChannels(channels)
		for _, channel := range channels {
			m.createChannelNode(node, channel)
		}
		node.Expand()
	}
	return nil
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

func yankID(node *tree.Node) tview.Cmd {
	if node == nil {
		return nil
	}

	id, ok := node.Reference().(fmt.Stringer)
	if !ok {
		return nil
	}
	return func() tview.Msg {
		if _, err := clipboard.Write(context.Background(), clipboard.FmtText, []byte(id.String())); err != nil {
			slog.Error("failed to write to clipboard", "err", err)
		}
		return nil
	}
}
