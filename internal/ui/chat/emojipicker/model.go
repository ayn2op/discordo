package emojipicker

import (
	"github.com/ayn2op/arikawa/v3/discord"
	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/discordo/internal/emoji"
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
	return Model{Model: uipicker.NewModel(cfg, "Add Reaction"), cfg: cfg}
}

var _ tview.Model[Model] = Model{}

// View shows the picker in a box titled Add Reaction.
func (m Model) View() tview.Widget {
	return m.Model.View(func(item picker.Item) tview.Msg {
		apiEmoji, ok := item.Reference.(discord.APIEmoji)
		if !ok {
			return nil
		}
		return SelectedMsg{Emoji: apiEmoji}
	}, CancelMsg{})
}

func (m Model) Update(msg tview.Msg) (Model, tview.Cmd) {
	var cmd tview.Cmd
	m.Model, cmd = m.Model.Update(msg)
	return m, cmd
}

// RefreshEmojis repopulates the picker with custom guild emojis and Unicode emojis.
func (m *Model) RefreshEmojis(state *ningen.State, guildID discord.GuildID) {
	var customEmojis []discord.Emoji
	if state != nil && state.EmojiState != nil {
		guilds, err := state.EmojiState.ForGuild(guildID)
		if err == nil {
			for _, g := range guilds {
				customEmojis = append(customEmojis, g.Emojis...)
			}
		}
	}

	results := emoji.Search("", customEmojis, 2500)
	items := make(picker.Items, 0, len(results))
	for _, item := range results {
		items = append(items, picker.Item{
			Text:       item.Display,
			FilterText: item.Name,
			Reference:  item.APIEmoji(),
		})
	}
	m.SetItems(items)
}
