// Package emojis completes emojis.
package emojis

import (
	"github.com/ayn2op/arikawa/v3/discord"
	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/discordo/internal/ui/autocomplete"
	uiemojis "github.com/ayn2op/discordo/internal/ui/emojis"
	"github.com/ayn2op/discordo/internal/ui/images"
	"github.com/ayn2op/ningen/v3"
	"github.com/ayn2op/tview"
	tviewimage "github.com/ayn2op/tview/image"
	"github.com/ayn2op/tview/text"
	"github.com/sahilm/fuzzy"
)

// Title is the autocomplete's title for emojis.
const Title = "Emojis"

type item struct {
	emoji discord.Emoji
	name  string
	style tview.Style
	// preview reports whether the image of a custom emoji is shown, so that none loaded for other panes is drawn when images are off.
	preview bool
}

func (i item) Insert() string { return i.emoji.String() }

func (i item) Line(images map[discord.URL]tviewimage.Widget) []text.Span {
	if !i.preview {
		images = nil
	}
	return uiemojis.Line(i.emoji, i.name, i.style, images)
}

// Items returns up to limit emojis matching query, best first, each once, and the command that loads their images, or nil.
func Items(cfg *config.Config, s *ningen.State, gID discord.GuildID, query string, limit int) ([]autocomplete.Item, tview.Cmd) {
	var names []string
	var emojis []discord.Emoji
	for name, emoji := range uiemojis.All(s, gID) {
		names, emojis = append(names, name), append(emojis, emoji)
	}

	matches := fuzzy.Find(query, names)
	// fuzzy matches nothing for an empty query.
	if query == "" {
		for i, name := range names {
			matches = append(matches, fuzzy.Match{Str: name, Index: i})
		}
	}

	preview := cfg.UI.Autocomplete.Emojis.Preview
	var items []autocomplete.Item
	load := images.RequestedMsg{Squares: map[discord.URL]uint{}}
	shown := make(map[discord.APIEmoji]bool)
	for _, match := range matches {
		if len(items) == limit {
			break
		}

		// An emoji has aliases.
		emoji := emojis[match.Index]
		if shown[emoji.APIString()] {
			continue
		}
		shown[emoji.APIString()] = true

		if preview && !emoji.IsUnicode() {
			load.Squares[uiemojis.URLFromID(emoji.ID)] = uiemojis.Rows
		}
		items = append(items, item{emoji, cfg.UI.MessagesList.Emojis.Name(match.Str), cfg.Theme.MessagesList.EmojiStyle.Style, preview})
	}

	if len(load.Squares) == 0 {
		return items, nil
	}
	return items, func() tview.Msg { return load }
}
