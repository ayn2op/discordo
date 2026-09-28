package channelspicker

import (
	"log/slog"
	"strings"

	"github.com/ayn2op/arikawa/v3/discord"
	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/discordo/internal/ui"
	uipicker "github.com/ayn2op/discordo/internal/ui/picker"
	"github.com/ayn2op/ningen/v3"
	"github.com/ayn2op/tview"
	"github.com/ayn2op/tview/picker"
)

type Model struct {
	uipicker.Model
	cfg *config.Config
}

func NewModel(cfg *config.Config) Model {
	return Model{Model: uipicker.NewModel(cfg, "Channels"), cfg: cfg}
}

var _ tview.Model[Model] = Model{}

// View shows the picker in a box titled Channels.
func (m Model) View() tview.Element {
	return m.Model.View(func(item picker.Item) tview.Msg {
		channelID, ok := item.Reference.(discord.ChannelID)
		if !ok || !channelID.IsValid() {
			return nil
		}
		return SelectedMsg{ChannelID: channelID}
	}, CancelMsg{})
}

func (m Model) Update(msg tview.Msg) (Model, tview.Cmd) {
	var cmd tview.Cmd
	m.Model, cmd = m.Model.Update(msg)
	return m, cmd
}

func (m *Model) RefreshChannels(state *ningen.State) {
	var items picker.Items

	privateChannels, err := state.Cabinet.PrivateChannels()
	if err != nil {
		slog.Error("failed to get private channels from state", "err", err)
		return
	}

	ui.SortPrivateChannels(privateChannels)
	for _, channel := range privateChannels {
		items = append(items, m.channelItem(state, nil, channel))
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
			items = append(items, m.channelItem(state, &guild, channel))
		}
	}

	m.SetItems(items)
}

func (m Model) channelItem(state *ningen.State, guild *discord.Guild, channel discord.Channel) picker.Item {
	var b strings.Builder
	b.WriteString(ui.ChannelToString(channel, m.cfg.Icons, state))

	if guild != nil {
		b.WriteString(" - ")
		b.WriteString(guild.Name)
	}

	name := b.String()
	return picker.Item{Text: name, FilterText: name, Reference: channel.ID}
}
