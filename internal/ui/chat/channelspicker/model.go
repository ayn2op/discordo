package channelspicker

import (
	"log/slog"
	"strings"

	"github.com/ayn2op/arikawa/v3/discord"
	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/discordo/internal/ui"
	"github.com/ayn2op/ningen/v3"
	"github.com/ayn2op/tview"
	"github.com/ayn2op/tview/picker"
)

type Model struct {
	items       picker.Items
	searchState picker.SearchState
	cfg         *config.Config
}

func NewModel(cfg *config.Config) Model {
	return Model{searchState: picker.NewSearchState(), cfg: cfg}
}

var _ tview.Model[Model] = Model{}

// changeMsg changes the picker.
type changeMsg picker.Change

func (Model) Init() tview.Cmd { return nil }

// View shows the picker in a box titled Channels.
func (m Model) View() tview.Element {
	p := ui.Picker(m.items, &m.searchState, m.cfg).
		OnChange(func(a picker.Change) tview.Msg { return changeMsg(a) }).
		OnSelect(func(item picker.Item) tview.Msg {
			channelID, ok := item.Reference.(discord.ChannelID)
			if !ok || !channelID.IsValid() {
				return nil
			}
			return SelectedMsg{ChannelID: channelID}
		}).
		OnCancel(CancelMsg{})
	return ui.Box(p, &m.cfg.Theme, true).Title("Channels")
}

func (m Model) Update(msg tview.Msg) (Model, tview.Cmd) {
	if msg, ok := msg.(changeMsg); ok {
		m.searchState.Apply(picker.Change(msg))
	}
	return m, nil
}

// Reset clears the query.
func (m *Model) Reset() { m.searchState.Reset() }

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

	m.items = items
	m.searchState.Reset()
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
