package messageslist

import (
	"image"
	"slices"

	"github.com/ayn2op/arikawa/v3/discord"
	md "github.com/ayn2op/arikawa/v3/markdown"
	"github.com/ayn2op/discordo/internal/ui/emojis"
	"github.com/ayn2op/discordo/internal/ui/images"
	"github.com/ayn2op/tview"
	"github.com/ayn2op/tview/text"
	"github.com/yuin/goldmark/v2/ast"
)

type previewSource struct {
	proxy         discord.URL
	width, height uint
}

func (m Model) previewSources(message discord.Message) []previewSource {
	var sources []previewSource
	for _, attachment := range message.Attachments {
		sources = append(sources, previewSource{attachment.Proxy, attachment.Width, attachment.Height})
	}

	embeds := message.Embeds
	if !m.cfg.UI.MessagesList.Embeds.Enabled {
		embeds = nil
	}
	for _, embed := range embeds {
		switch {
		case embed.Image != nil:
			sources = append(sources, previewSource{embed.Image.Proxy, embed.Image.Width, embed.Image.Height})
		// Image and GIF embeds carry the media itself as the thumbnail.
		case embed.Thumbnail != nil && (embed.Type == discord.ImageEmbed || embed.Type == discord.GIFVEmbed):
			sources = append(sources, previewSource{embed.Thumbnail.Proxy, embed.Thumbnail.Width, embed.Thumbnail.Height})
		}
	}

	// Non-image files have no dimensions.
	return slices.DeleteFunc(sources, func(source previewSource) bool { return source.width == 0 || source.height == 0 })
}

// LoadImages returns the command that loads the images of the shown messages, or nil.
func (m Model) LoadImages() tview.Cmd {
	msg := images.RequestedMsg{Squares: map[discord.URL]uint{}, Previews: map[discord.URL]image.Point{}}
	for index := range m.items {
		item := &m.items[index]
		if item.separator != nil {
			continue
		}

		if author := item.message.Author; m.showsAvatar(author) {
			msg.Squares[avatarURL(author)] = uint(m.cfg.UI.MessagesList.Avatars.Height)
		}

		m.addEmojis(msg.Squares, item)
		if !m.cfg.UI.MessagesList.Attachments.Preview {
			continue
		}

		for _, source := range m.previewSources(item.message) {
			msg.Previews[source.proxy] = image.Pt(int(source.width), int(source.height))
		}
	}

	if len(msg.Squares) == 0 && len(msg.Previews) == 0 {
		return nil
	}

	return func() tview.Msg { return msg }
}

// addEmojis adds the custom emoji of the reactions to the item's message and in its content to squares.
func (m Model) addEmojis(squares map[discord.URL]uint, item *messageItem) {
	for _, reaction := range item.message.Reactions {
		if m.cfg.UI.MessagesList.Reactions.Enabled && reaction.Emoji.IsCustom() {
			squares[emojis.URLFromID(reaction.Emoji.ID)] = emojis.Rows
		}
	}

	root := m.contentRoot(item)
	if root == nil {
		return
	}

	_ = ast.Walk(root, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		emoji, ok := node.(*md.Emoji)
		if !ok || !entering {
			return ast.WalkContinue, nil
		}

		if id, err := discord.ParseSnowflake(emoji.ID); err == nil {
			squares[emojis.URLFromID(discord.EmojiID(id))] = emojis.Rows
		}

		return ast.WalkContinue, nil
	})
}

// showsAvatar reports whether messages by user are drawn beside their avatar.
func (m Model) showsAvatar(user discord.User) bool {
	avatars := m.cfg.UI.MessagesList.Avatars
	return avatars.Preview && avatars.Height != 0 && !m.blocked(user)
}

func avatarURL(user discord.User) discord.URL {
	return user.AvatarURLWithType(discord.PNGImage) + "?size=128"
}

// emojiLine returns the image of the custom emoji with id as a line of spans, or nil if it is not loaded.
func (m Model) emojiLine(id string) []text.Span {
	snowflake, err := discord.ParseSnowflake(id)
	if err != nil {
		return nil
	}

	emoji, ok := m.images[emojis.URLFromID(discord.EmojiID(snowflake))]
	if !ok {
		return nil
	}

	if lines := emoji.Lines(); len(lines) > 0 {
		return lines[0]
	}

	return nil
}
