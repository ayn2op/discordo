package chat

import (
	"log/slog"
	"strings"
	"reflect"

	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/discordo/internal/ui"
	"github.com/ayn2op/discordo/pkg/picker"
	"github.com/ayn2op/tview"
	"github.com/diamondburned/arikawa/v3/discord"
)

type channelsPicker struct {
	*picker.Picker
	chatView *View
}

func newChannelsPicker(cfg *config.Config, chatView *View) *channelsPicker {
	cp := &channelsPicker{picker.New(), chatView}
	cp.Box = ui.ConfigureBox(tview.NewBox(), &cfg.Theme)
	// When a child of tview.Flex is focused, tview.Flex itself is not reported as focused. Instead, the focused child (picker) is considered focused. Therefore, we manually set the active border style on the picker to ensure it displays the correct focused appearance.
	cp.
		SetBlurFunc(nil).
		SetFocusFunc(nil).
		SetBorderSet(cfg.Theme.Border.ActiveSet.BorderSet).
		SetBorderStyle(cfg.Theme.Border.ActiveStyle.Style).
		SetTitleStyle(cfg.Theme.Title.ActiveStyle.Style).
		SetFooterStyle(cfg.Theme.Footer.ActiveStyle.Style)

	cp.SetSelectedFunc(cp.onSelected)
	cp.SetTitle("Channels")
	cp.SetKeyMap(&picker.KeyMap{
		Cancel: cfg.Keybinds.Picker.Cancel,
		Up:     cfg.Keybinds.Picker.Up,
		Down:   cfg.Keybinds.Picker.Down,
		Top:    cfg.Keybinds.Picker.Top,
		Bottom: cfg.Keybinds.Picker.Bottom,
		Select: cfg.Keybinds.Picker.Select,
	})
	return cp
}

func (cp *channelsPicker) onSelected(item picker.Item) {
	channelID, ok := item.Reference.(discord.ChannelID)
	if !ok || !channelID.IsValid() {
		return
	}

	channel, err := cp.chatView.state.Cabinet.Channel(channelID)
	if err != nil {
		slog.Error("failed to get channel from state", "err", err, "channel_id", channelID)
		return
	}

	node := cp.chatView.guildsTree.findNodeByChannelID(channel.ID)
	if node == nil {
		slog.Error("failed to locate channel in tree", "channel_id", channel.ID)
		return
	}

	cp.chatView.guildsTree.expandPathToNode(node)
	cp.chatView.guildsTree.SetCurrentNode(node)
	if channel.Type != discord.GuildCategory {
		cp.chatView.guildsTree.onSelected(node)
	}
	cp.chatView.closePicker()
	cp.chatView.focusMessageInput()
}

func (cp *channelsPicker) update() {
	cp.ClearItems()
	state := cp.chatView.state

	privateChannels, err := state.Cabinet.PrivateChannels()
	if err != nil {
		slog.Error("failed to get private channels from state", "err", err)
		return
	}

	ui.SortPrivateChannels(privateChannels)
	for _, channel := range privateChannels {
		cp.addChannel(nil, channel)
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
			cp.addChannel(&guild, channel)
		}
	}

	cp.Update()
}

func (cp *channelsPicker) addChannel(guild *discord.Guild, channel discord.Channel) {
	var b strings.Builder
	b.WriteString(ui.ChannelToString(channel, cp.chatView.cfg.Icons))

	if guild != nil {
		b.WriteString(" - ")
		b.WriteString(guild.Name)
	}

	name := b.String()
	cp.AddItem(picker.Item{Text: name, FilterText: name, Reference: channel.ID})
}

// Set hotkeys on focus.
func (cp *channelsPicker) Focus(delegate func(p tview.Primitive)) {
	cp.chatView.hotkeysBar.hotkeysFromValue(
		reflect.ValueOf(cp.chatView.cfg.Keybinds.Picker),
		nil,
	)
	cp.Picker.Focus(delegate)
}
