package reactionspicker

import (
	"strings"

	"github.com/ayn2op/arikawa/v3/discord"
	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/discordo/internal/ui/emojis"
	"github.com/ayn2op/discordo/internal/ui/images"
	uipicker "github.com/ayn2op/discordo/internal/ui/picker"
	"github.com/ayn2op/ningen/v3"
	"github.com/ayn2op/tview"
	tviewimage "github.com/ayn2op/tview/image"
	"github.com/ayn2op/tview/picker"
	"github.com/ayn2op/tview/text"
)

type Model struct {
	uipicker.Model
	cfg   *config.Config
	state *ningen.State
	// reactions are those of the items, and images the chat model's, which loads them, by URL.
	reactions []reaction
	images    map[discord.URL]tviewimage.Widget
	// channelID and messageID are of the message reacted to.
	channelID discord.ChannelID
	messageID discord.MessageID
}

// reaction is an emoji to react with and its names, the first of which it is shown under.
type reaction struct {
	emoji discord.Emoji
	names []string
}

func NewModel(cfg *config.Config, state *ningen.State) Model {
	return Model{Model: uipicker.NewModel(cfg, "Reactions"), cfg: cfg, state: state}
}

// items returns the reactions as the picker's items, drawn with the images and found by any of their names.
func (m Model) items() picker.Items {
	style := m.cfg.Theme.MessagesList.EmojiStyle.Style
	items := make(picker.Items, len(m.reactions))
	for i, r := range m.reactions {
		line := emojis.Line(r.emoji, m.cfg.UI.MessagesList.Emojis.Name(r.names[0]), style, m.images)
		items[i] = picker.Item{Content: text.Rich(line...).Wrapping(text.WrappingNone), FilterText: strings.Join(r.names, " "), Reference: r.emoji.APIString()}
	}
	return items
}

var _ tview.Model[Model] = Model{}

func (m Model) Update(msg tview.Msg) (Model, tview.Cmd) {
	switch msg := msg.(type) {
	case ShowMsg:
		m.channelID, m.messageID, m.reactions = msg.ChannelID, msg.MessageID, nil
		load := images.RequestedMsg{Squares: map[discord.URL]uint{}}
		for name, emoji := range emojis.All(m.state, msg.GuildID) {
			// An emoji's names come together.
			if last := len(m.reactions) - 1; last >= 0 && m.reactions[last].emoji.APIString() == emoji.APIString() {
				m.reactions[last].names = append(m.reactions[last].names, name)
				continue
			}

			m.reactions = append(m.reactions, reaction{emoji, []string{name}})
			if !emoji.IsUnicode() {
				load.Squares[emojis.URLFromID(emoji.ID)] = emojis.Rows
			}
		}

		m.SetItems(m.items())
		if len(load.Squares) == 0 {
			return m, nil
		}
		return m, func() tview.Msg { return load }
	case images.ChangedMsg:
		m.images = msg
		m.ReplaceItems(m.items())
		return m, nil
	}

	var cmd tview.Cmd
	m.Model, cmd = m.Model.Update(msg)
	return m, cmd
}

// View shows the picker in a box titled Reactions.
func (m Model) View() tview.Widget {
	return m.Model.View(func(item picker.Item) tview.Msg {
		emoji, ok := item.Reference.(discord.APIEmoji)
		if !ok {
			return nil
		}
		return SelectedMsg{ChannelID: m.channelID, MessageID: m.messageID, Emoji: emoji}
	}, CancelMsg{})
}
