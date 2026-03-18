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
	chatView *Model
}

var _ help.KeyMap = (*channelsPicker)(nil)

func newChannelsPicker(cfg *config.Config, chatView *Model) *channelsPicker {
	cp := &channelsPicker{picker.NewModel(), chatView}
	ConfigurePicker(cp.Model, cfg, "Channels")
	return cp
}

func (cp *channelsPicker) HandleEvent(event tview.Event) tview.Command {
	switch event := event.(type) {
	case *picker.SelectedEvent:
		channelID, ok := event.Reference.(discord.ChannelID)
		if !ok || !channelID.IsValid() {
			return nil
		}

		channel, err := cp.chatView.state.Cabinet.Channel(channelID)
		if err != nil {
			slog.Error("failed to get channel from state", "err", err, "channel_id", channelID)
			return nil
		}

		node := cp.chatView.guildsTree.findNodeByChannelID(channel.ID)
		if node == nil {
			slog.Error("failed to locate channel in tree", "channel_id", channel.ID)
			return nil
		}

		cp.chatView.guildsTree.expandPathToNode(node)
		cp.chatView.guildsTree.SetCurrentNode(node)
		if channel.Type != discord.GuildCategory {
			cp.chatView.guildsTree.onSelected(node)
		}
		cp.chatView.closePicker()
		cp.chatView.focusMessageInput()
		return nil
	case *picker.CancelEvent:
		cp.chatView.closePicker()
		return nil
	}
	return cp.Model.HandleEvent(event)
}

func (cp *channelsPicker) update() {
	var items picker.Items
	state := cp.chatView.state

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

	cp.Model.SetItems(items)
}

func (cp *channelsPicker) channelItem(guild *discord.Guild, channel discord.Channel) picker.Item {
	var b strings.Builder
	b.WriteString(ui.ChannelToString(channel, cp.chatView.cfg.Icons, cp.chatView.state))

	if guild != nil {
		b.WriteString(" - ")
		b.WriteString(guild.Name)
	}

	name := b.String()
	return picker.Item{Text: name, FilterText: name, Reference: channel.ID}
}

func (cp *channelsPicker) ShortHelp() []keybind.Keybind {
	cfg := cp.chatView.cfg.Keybinds.Picker
	return []keybind.Keybind{cfg.Up.Keybind, cfg.Down.Keybind, cfg.Select.Keybind, cfg.Cancel.Keybind}
}

func (cp *channelsPicker) FullHelp() [][]keybind.Keybind {
	cfg := cp.chatView.cfg.Keybinds.Picker
	return [][]keybind.Keybind{
		{cfg.Up.Keybind, cfg.Down.Keybind, cfg.Top.Keybind, cfg.Bottom.Keybind},
		{cfg.Select.Keybind, cfg.Cancel.Keybind},
	}
}
