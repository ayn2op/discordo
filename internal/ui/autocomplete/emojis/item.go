// Package emojis completes emojis.
package emojis

import (
	"slices"

	"github.com/ayn2op/arikawa/v3/discord"
	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/discordo/internal/ui"
	"github.com/ayn2op/discordo/internal/ui/autocomplete"
	"github.com/ayn2op/discordo/internal/ui/images"
	"github.com/ayn2op/discordo/pkg/discordemoji"
	"github.com/ayn2op/ningen/v3"
	"github.com/ayn2op/tview"
	tviewimage "github.com/ayn2op/tview/image"
	"github.com/ayn2op/tview/text"
	"github.com/sahilm/fuzzy"
)

const (
	// Title is the autocomplete's title for emojis.
	Title     = "Emojis"
	separator = " "
)

type customEmoji struct {
	emoji discord.Emoji
	name  string
	// image is empty if images are off, so none loaded for other panes is drawn.
	image discord.URL
	style tview.Style
}

func (e customEmoji) Insert() string { return e.emoji.String() }

func (e customEmoji) Line(images map[discord.URL]tviewimage.Widget) []text.Span {
	line := []text.Span{{Text: e.name, Style: e.style}}
	if image, ok := images[e.image]; ok {
		if lines := image.Lines(); len(lines) > 0 {
			line = slices.Concat(lines[0], []text.Span{{Text: separator}}, line)
		}
	}
	return line
}

type unicodeEmoji struct{ emoji, name string }

func (e unicodeEmoji) Insert() string { return e.emoji }

func (e unicodeEmoji) Line(map[discord.URL]tviewimage.Widget) []text.Span {
	return []text.Span{{Text: e.emoji + separator + e.name}}
}

// Items returns up to limit emojis matching query, best first, each once, and the command that loads their images, or nil.
func Items(cfg *config.Config, s *ningen.State, gID discord.GuildID, query string, limit int) ([]autocomplete.Item, tview.Cmd) {
	custom := usableEmojis(s, gID)
	var names []string
	for _, emoji := range custom {
		names = append(names, emoji.Name)
	}

	var unicode []unicodeEmoji
	for name, emoji := range discordemoji.All() {
		names = append(names, name)
		unicode = append(unicode, unicodeEmoji{emoji, name})
	}

	matches := fuzzy.Find(query, names)
	// fuzzy matches nothing for an empty query.
	if query == "" {
		for i, name := range names {
			matches = append(matches, fuzzy.Match{Str: name, Index: i})
		}
	}

	emojisCfg := cfg.UI.MessagesList.Emojis
	var items []autocomplete.Item
	images := images.RequestedMsg{Squares: map[discord.URL]uint{}}
	shown := make(map[string]bool)
	for _, match := range matches {
		if len(items) == limit {
			break
		}

		name := emojisCfg.Name(match.Str)
		if match.Index < len(custom) {
			emoji := customEmoji{custom[match.Index], name, "", cfg.Theme.MessagesList.EmojiStyle.Style}
			if cfg.UI.Autocomplete.Emojis.Preview {
				emoji.image = emoji.emoji.EmojiURLWithType(discord.PNGImage)
				images.Squares[emoji.image] = ui.EmojiRows
			}

			items = append(items, emoji)
			continue
		}

		// An emoji has aliases.
		emoji := unicode[match.Index-len(custom)].emoji
		if !shown[emoji] {
			shown[emoji] = true
			items = append(items, unicodeEmoji{emoji, name})
		}
	}

	if len(images.Squares) == 0 {
		return items, nil
	}
	return items, func() tview.Msg { return images }
}

// usableEmojis returns the custom emojis the user may send in the guild, its own first.
func usableEmojis(s *ningen.State, gID discord.GuildID) []discord.Emoji {
	me, err := s.Cabinet.Me()
	if err != nil {
		return nil
	}

	nitro := me.Nitro != discord.NoUserNitro
	gIDs := []discord.GuildID{gID}
	if guilds, err := s.Cabinet.Guilds(); nitro && err == nil {
		for _, guild := range guilds {
			if guild.ID != gID {
				gIDs = append(gIDs, guild.ID)
			}
		}
	}

	var usable []discord.Emoji
	for _, gID := range gIDs {
		emojis, err := s.Cabinet.Emojis(gID)
		if err != nil {
			continue
		}

		var roles []discord.RoleID
		if member, err := s.Cabinet.Member(gID, me.ID); err == nil {
			roles = member.RoleIDs
		}

		for _, emoji := range emojis {
			if (nitro || !emoji.Animated) && isAllowed(emoji, roles) {
				usable = append(usable, emoji)
			}
		}
	}

	return usable
}

// isAllowed reports whether a member with roles may use emoji.
func isAllowed(emoji discord.Emoji, roles []discord.RoleID) bool {
	return len(emoji.RoleIDs) == 0 || slices.ContainsFunc(emoji.RoleIDs, func(id discord.RoleID) bool {
		return slices.Contains(roles, id)
	})
}
