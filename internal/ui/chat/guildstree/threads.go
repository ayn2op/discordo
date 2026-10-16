package guildstree

import (
	"log/slog"
	"slices"

	"github.com/ayn2op/arikawa/v3/discord"
	"github.com/ayn2op/arikawa/v3/gateway"
)

func (m Model) upsertThread(thread discord.Channel) {
	if !isThread(thread.Type) || !thread.ParentID.IsValid() {
		return
	}
	m.removeThread(thread.ID, thread.ParentID)
	if thread.ThreadMetadata != nil && thread.ThreadMetadata.Archived {
		return
	}
	if parent := m.findNodeByReference(thread.ParentID); parent != nil {
		m.createChannelNode(parent, thread)
	}
}

func (m Model) removeThread(id, parentID discord.ChannelID) {
	if node := m.findNodeByReference(id); node != nil {
		if parent := m.findNodeByReference(parentID); parent != nil {
			parent.RemoveChild(node)
		}
	}
	delete(m.nodes, discord.Snowflake(id))
}

func (m Model) syncThreads(event *gateway.ThreadListSyncEvent) {
	channels, err := m.state.Cabinet.Channels(event.GuildID)
	if err != nil {
		slog.Error("failed to get channels from state", "err", err, "guild_id", event.GuildID)
		return
	}

	for _, thread := range channels {
		if !isThread(thread.Type) || !thread.ParentID.IsValid() {
			continue
		}
		if event.ChannelIDs == nil || slices.Contains(event.ChannelIDs, thread.ParentID) {
			m.removeThread(thread.ID, thread.ParentID)
		}
	}

	for _, thread := range event.Threads {
		m.upsertThread(thread)
	}
}
