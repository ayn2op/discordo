// Package emojis is the emojis the user may send, and how custom ones are drawn with the same images in every pane.
package emojis

import (
	"iter"
	"slices"

	"github.com/ayn2op/arikawa/v3/discord"
	"github.com/ayn2op/discordo/pkg/discordemoji"
	"github.com/ayn2op/ningen/v3"
	"github.com/ayn2op/tview"
	tviewimage "github.com/ayn2op/tview/image"
	"github.com/ayn2op/tview/text"
)

// Rows is how tall custom emojis are drawn, the same in every pane, as they share the images by URL.
const Rows = 1

// URLFromID returns the URL of the image of the custom emoji with id, the same in every pane.
func URLFromID(id discord.EmojiID) discord.URL {
	return discord.Emoji{ID: id}.EmojiURLWithType(discord.PNGImage)
}

// All yields the emojis the user may send in the guild under each of their names: the custom ones first, then the Unicode ones, each emoji's names together.
func All(s *ningen.State, gID discord.GuildID) iter.Seq2[string, discord.Emoji] {
	return func(yield func(string, discord.Emoji) bool) {
		for _, emoji := range usableEmojis(s, gID) {
			if !yield(emoji.Name, emoji) {
				return
			}
		}
		for name, emoji := range discordemoji.All() {
			if !yield(name, discord.Emoji{Name: emoji}) {
				return
			}
		}
	}
}

const separator = " "

// Line returns emoji as drawn under name: a custom one in style after its image, if images has it, and a Unicode one before name.
func Line(emoji discord.Emoji, name string, style tview.Style, images map[discord.URL]tviewimage.Widget) []text.Span {
	if emoji.IsUnicode() {
		return []text.Span{{Text: emoji.Name + separator + name}}
	}

	line := []text.Span{{Text: name, Style: style}}
	if image, ok := images[URLFromID(emoji.ID)]; ok {
		if lines := image.Lines(); len(lines) > 0 {
			line = slices.Concat(lines[0], []text.Span{{Text: separator}}, line)
		}
	}
	return line
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
