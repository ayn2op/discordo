// Package inbox lists the channels with unread messages, so that one is opened or marked read without being looked for in the tree.
package inbox

import (
	"cmp"
	"log/slog"
	"slices"
	"strconv"

	"github.com/ayn2op/arikawa/v3/discord"
	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/discordo/internal/ui"
	uipicker "github.com/ayn2op/discordo/internal/ui/picker"
	"github.com/ayn2op/ningen/v3"
	"github.com/ayn2op/tview"
	"github.com/ayn2op/tview/keybind"
	"github.com/ayn2op/tview/picker"
)

// entry is the unread channel a row stands for.
type entry struct {
	channelID discord.ChannelID
	// lastMessageID is the message that acknowledging the channel marks read up to.
	lastMessageID discord.MessageID
	// lastReadID is the message read last, which the first unread one follows. Opening the channel acknowledges them all, so it is kept from before that.
	lastReadID discord.MessageID
	// mentions is how many of the unread messages mention the user, which puts the row before those with fewer.
	mentions int
	// text is the row as shown: the channel, the guild it is in, and the mentions it has.
	text string
}

type Model struct {
	uipicker.Model
	cfg   *config.Config
	state *ningen.State
	// entries are the rows listed, which marking all read acknowledges.
	entries []entry
}

func NewModel(cfg *config.Config, state *ningen.State) Model {
	return Model{Model: uipicker.NewModel(cfg, "Inbox"), cfg: cfg, state: state}
}

var _ tview.Model[Model] = Model{}

// View shows the picker in a box titled Inbox.
func (m Model) View() tview.Widget {
	return m.Model.View(func(item picker.Item) tview.Msg {
		selected, ok := item.Reference.(entry)
		if !ok || !selected.channelID.IsValid() {
			return nil
		}
		return SelectedMsg{ChannelID: selected.channelID, LastReadID: selected.lastReadID}
	}, CancelMsg{})
}

func (m Model) Update(msg tview.Msg) (Model, tview.Cmd) {
	if key, ok := msg.(tview.KeyMsg); ok {
		kbs := m.cfg.Keybinds.Inbox
		switch {
		case keybind.Matches(key, kbs.MarkRead.Keybind):
			return m, m.markRead()
		case keybind.Matches(key, kbs.MarkAllRead.Keybind):
			return m, m.markAllRead()
		}
	}

	var cmd tview.Cmd
	m.Model, cmd = m.Model.Update(msg)
	return m, cmd
}

// Refresh lists the channels that are unread, those mentioning the user first and the ones written to last next.
func (m *Model) Refresh() {
	var entries []entry
	entries = m.appendPrivateChannels(entries)
	entries = m.appendGuildChannels(entries)

	// Mentions are what the inbox is opened for, so they come first, and the newest message decides the order of the rest.
	slices.SortFunc(entries, func(a, b entry) int {
		if order := cmp.Compare(b.mentions, a.mentions); order != 0 {
			return order
		}
		return cmp.Compare(b.lastMessageID, a.lastMessageID)
	})

	m.setEntries(entries)
}

// appendPrivateChannels appends the unread direct messages.
func (m Model) appendPrivateChannels(entries []entry) []entry {
	channels, err := m.state.Cabinet.PrivateChannels()
	if err != nil {
		slog.Error("failed to get private channels from state", "err", err)
		return entries
	}

	for _, channel := range channels {
		entries = m.appendUnread(entries, channel, "")
	}

	return entries
}

// appendGuildChannels appends the unread channels of every guild, each named after the guild it is in.
func (m Model) appendGuildChannels(entries []entry) []entry {
	guilds, err := m.state.Cabinet.Guilds()
	if err != nil {
		slog.Error("failed to get guilds from state", "err", err)
		return entries
	}

	for _, guild := range guilds {
		channels, err := m.state.Cabinet.Channels(guild.ID)
		if err != nil {
			slog.Error("failed to get channels from state", "err", err, "guild_id", guild.ID)
			continue
		}

		for _, channel := range channels {
			entries = m.appendUnread(entries, channel, guild.Name)
		}
	}

	return entries
}

// appendUnread appends channel, in the guild named guildName or none of one for a direct message, unless it is read.
// A muted channel reads as read unless it mentions the user, which leaves it out as it is in Discord's inbox.
func (m Model) appendUnread(entries []entry, channel discord.Channel, guildName string) []entry {
	if m.state.ChannelIsUnread(channel.ID, ningen.UnreadOpts{}) == ningen.ChannelRead {
		return entries
	}

	text := ui.ChannelToString(channel, m.cfg.Icons, m.state)
	if guildName != "" {
		text += " - " + guildName
	}

	var mentions int
	var lastReadID discord.MessageID
	if read := m.state.ReadState.ReadState(channel.ID); read != nil {
		mentions, lastReadID = read.MentionCount, read.LastMessageID
	}

	// The count follows the name, as the badge does in Discord: "# general - My Guild (3)".
	if mentions > 0 {
		text += " (" + strconv.Itoa(mentions) + ")"
	}

	return append(entries, entry{
		channelID:     channel.ID,
		lastMessageID: m.state.LastMessage(channel.ID),
		lastReadID:    lastReadID,
		mentions:      mentions,
		text:          text,
	})
}

// setEntries shows a row per entry. The query typed is dropped, as the matches it left refer to the rows replaced.
func (m *Model) setEntries(entries []entry) {
	m.entries = entries

	items := make(picker.Items, len(entries))
	for index, e := range entries {
		items[index] = picker.Item{Text: e.text, FilterText: e.text, Reference: e}
	}

	m.SetItems(items)
}

// markRead acknowledges the highlighted channel and drops its row, as opening it would.
func (m *Model) markRead() tview.Cmd {
	item, ok := m.Selected()
	if !ok {
		return nil
	}

	selected, ok := item.Reference.(entry)
	if !ok {
		return nil
	}

	m.setEntries(slices.DeleteFunc(slices.Clone(m.entries), func(e entry) bool { return e.channelID == selected.channelID }))
	return ack(m.state, selected)
}

// markAllRead acknowledges every channel listed and empties the inbox.
func (m *Model) markAllRead() tview.Cmd {
	cmds := make([]tview.Cmd, 0, len(m.entries))
	for _, e := range m.entries {
		cmds = append(cmds, ack(m.state, e))
	}

	m.setEntries(nil)
	return tview.Batch(cmds...)
}

// ack tells Discord that the messages of e's channel up to its last one were read.
func ack(state *ningen.State, e entry) tview.Cmd {
	return func() tview.Msg {
		state.ReadState.MarkRead(e.channelID, e.lastMessageID)
		return nil
	}
}
