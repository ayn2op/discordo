package chat

import (
	"log/slog"
	"strings"

	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/discordo/internal/ui"
	"github.com/ayn2op/tview"
	"github.com/ayn2op/tview/help"
	"github.com/ayn2op/tview/keybind"
	"github.com/ayn2op/tview/picker"
	"github.com/diamondburned/arikawa/v3/discord"
)

type channelsPicker struct {
	*picker.Model
	chat *Model
	cfg  *config.Config
}

var _ help.KeyMap = (*channelsPicker)(nil)

func newChannelsPicker(cfg *config.Config, chat *Model) *channelsPicker {
	cp := &channelsPicker{Model: picker.NewModel(), chat: chat, cfg: cfg}
	ConfigurePicker(cp.Model, cfg, "Channels")
	return cp
}

func (cp *channelsPicker) Update(msg tview.Msg) tview.Cmd {
	switch msg := msg.(type) {
	case picker.SelectedMsg:
		channelID, ok := msg.Reference.(discord.ChannelID)
		if !ok || !channelID.IsValid() {
			return nil
		}

		channel, err := cp.chat.state.Cabinet.Channel(channelID)
		if err != nil {
			slog.Error("failed to get channel from state", "err", err, "channel_id", channelID)
			return nil
		}

		node := cp.chat.guildsTree.findNodeByChannelID(channel.ID)
		if node == nil {
			slog.Error("failed to locate channel in tree", "channel_id", channel.ID)
			return nil
		}

		cp.chat.guildsTree.expandPathToNode(node)
		cp.chat.guildsTree.SetCurrentNode(node)
		var selectCmd tview.Cmd
		if channel.Type != discord.GuildCategory {
			selectCmd = cp.chat.guildsTree.onSelected(node)
		}
		cp.chat.closePicker()
		return selectCmd
	case picker.CancelMsg:
		cp.chat.closePicker()
		return nil
	}
	return cp.Model.Update(msg)
}

func (cp *channelsPicker) update() {
	var items picker.Items
	state := cp.chat.state

	privateChannels, err := state.Cabinet.PrivateChannels()
	if err != nil {
		slog.Error("failed to get private channels from state", "err", err)
		return
	}

	ui.SortPrivateChannels(privateChannels)
	for _, channel := range privateChannels {
		items = append(items, cp.channelItem(nil, channel))
	}

	guilds, err := state.Cabinet.Guilds()
	if err != nil {
		slog.Error("failed to get guilds from state", "err", err)
		return
	}

	for _, guild := range guilds {
		channels, err := state.Cabinet.Channels(guild.ID)
		if err != nil {
			slog.Error("failed to get channels from state", "err", err, "guild_id", guild.ID)
			continue
		}

		for _, channel := range channels {
			items = append(items, cp.channelItem(&guild, channel))
		}
	}

	cp.SetItems(items)
}

func (cp *channelsPicker) channelItem(guild *discord.Guild, channel discord.Channel) picker.Item {
	var b strings.Builder
	b.WriteString(ui.ChannelToString(channel, cp.cfg.Icons, cp.chat.state))

	if guild != nil {
		b.WriteString(" - ")
		b.WriteString(guild.Name)
	}

	name := b.String()
	return picker.Item{Text: name, FilterText: name, Reference: channel.ID}
}

func (cp *channelsPicker) ShortHelp() []keybind.Keybind {
	cfg := cp.cfg.Keybinds.Picker
	return []keybind.Keybind{cfg.Up.Keybind, cfg.Down.Keybind, cfg.Select.Keybind, cfg.Cancel.Keybind}
}

func (cp *channelsPicker) FullHelp() [][]keybind.Keybind {
	cfg := cp.cfg.Keybinds.Picker
	return [][]keybind.Keybind{
		{cfg.Up.Keybind, cfg.Down.Keybind, cfg.Top.Keybind, cfg.Bottom.Keybind},
		{cfg.Select.Keybind, cfg.Cancel.Keybind},
	}
}
