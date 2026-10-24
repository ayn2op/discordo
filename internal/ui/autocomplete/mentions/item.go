// Package mentions completes mentions.
package mentions

import (
	"cmp"
	"log/slog"
	"slices"

	"github.com/ayn2op/arikawa/v3/discord"
	"github.com/ayn2op/arikawa/v3/state"
	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/discordo/internal/ui"
	"github.com/ayn2op/discordo/internal/ui/autocomplete"
	"github.com/ayn2op/ningen/v3"
	"github.com/ayn2op/tview"
	tviewimage "github.com/ayn2op/tview/image"
	"github.com/ayn2op/tview/text"
	"github.com/gdamore/tcell/v3"
	"github.com/sahilm/fuzzy"
)

const (
	// Title is the autocomplete's title for mentions.
	Title     = "Mentions"
	separator = " "
	// avatarRows is one line, so avatars use their own URL instead of sharing the taller ones in the messages list.
	avatarRows = 1
	avatarSize = "?size=32"
)

type mention struct {
	username, name string
	avatar         discord.URL
	style          tview.Style
}

func (m mention) Insert() string { return "@" + m.username }

func (m mention) Line(images map[discord.URL]tviewimage.Widget) []text.Span {
	line := []text.Span{{Text: m.name, Style: m.style}}
	if image, ok := images[m.avatar]; ok {
		if lines := image.Lines(); len(lines) > 0 {
			line = slices.Concat(lines[0], []text.Span{{Text: separator}}, line)
		}
	}
	return line
}

// Items returns up to limit users in channel matching query, best first, or its recent authors if query is empty, and the command that loads their avatars, or nil.
func Items(cfg *config.Config, s *ningen.State, channel *discord.Channel, query string, limit int) ([]autocomplete.Item, tview.Cmd) {
	var items []autocomplete.Item
	images := ui.LoadImagesMsg{Squares: map[discord.URL]uint{}}
	for _, mention := range mentions(s, channel, query, limit) {
		items = append(items, mention)
		images.Squares[mention.avatar] = avatarRows
	}

	if !cfg.UI.Autocomplete.Mentions.Preview || len(images.Squares) == 0 {
		return items, nil
	}
	return items, func() tview.Msg { return images }
}

func mentions(s *ningen.State, channel *discord.Channel, query string, limit int) []mention {
	if query == "" {
		return recentAuthors(s, channel, limit)
	}

	var items []mention
	if !channel.GuildID.IsValid() {
		// DMs have recipients, not members.
		me, _ := s.Cabinet.Me()
		users := append(slices.Clone(channel.DMRecipients), *me)
		for _, r := range fuzzy.FindFrom(query, userList(users)) {
			if len(items) == limit {
				break
			}

			items = append(items, userMention(s, &users[r.Index]))
		}
		return items
	}

	members, err := s.Cabinet.Members(channel.GuildID)
	if err != nil {
		slog.Error("failed to get members from state", "guild_id", channel.GuildID, "err", err)
		return nil
	}

	for _, r := range fuzzy.FindFrom(query, memberList(members)) {
		if len(items) == limit {
			break
		}

		if ChannelHasUser(s, channel.ID, members[r.Index].User.ID) {
			items = append(items, memberMention(s, channel.GuildID, &members[r.Index]))
		}
	}
	return items
}

func recentAuthors(s *ningen.State, channel *discord.Channel, limit int) []mention {
	messages, err := s.Cabinet.Messages(channel.ID)
	if err != nil {
		return nil
	}

	me, _ := s.Cabinet.Me()
	shown := map[discord.UserID]bool{me.ID: true}
	var items []mention
	for _, message := range messages {
		if len(items) == limit {
			break
		}

		if shown[message.Author.ID] {
			continue
		}

		shown[message.Author.ID] = true
		if !channel.GuildID.IsValid() {
			items = append(items, userMention(s, &message.Author))
			continue
		}

		s.MemberState.RequestMember(channel.GuildID, message.Author.ID)
		if member, err := s.Cabinet.Member(channel.GuildID, message.Author.ID); err == nil {
			items = append(items, memberMention(s, channel.GuildID, member))
		}
	}
	return items
}

func memberMention(s *ningen.State, gID discord.GuildID, member *discord.Member) mention {
	style := presenceStyle(s, gID, member.User.ID)
	// This avoids a slower member color lookup path.
	color, ok := state.MemberColor(member, func(id discord.RoleID) *discord.Role {
		r, _ := s.Cabinet.Role(gID, id)
		return r
	})
	if ok {
		style = style.Foreground(tcell.NewHexColor(int32(color)))
	}

	return mention{member.User.Username, cmp.Or(member.Nick, member.User.DisplayName, member.User.Username), avatarURL(member.User), style}
}

func userMention(s *ningen.State, user *discord.User) mention {
	return mention{user.Username, user.DisplayOrUsername(), avatarURL(*user), presenceStyle(s, discord.NullGuildID, user.ID)}
}

func avatarURL(user discord.User) discord.URL {
	return user.AvatarURLWithType(discord.PNGImage) + avatarSize
}

// presenceStyle dims offline users.
func presenceStyle(s *ningen.State, gID discord.GuildID, userID discord.UserID) tview.Style {
	presence, err := s.Cabinet.Presence(gID, userID)
	if err != nil {
		slog.Info("failed to get presence from state", "guild_id", gID, "user_id", userID, "err", err)
	}
	return tcell.StyleDefault.Dim(err == nil && presence.Status == discord.OfflineStatus)
}

// ChannelHasUser reports whether the user may view the channel.
func ChannelHasUser(s *ningen.State, channelID discord.ChannelID, userID discord.UserID) bool {
	perms, err := s.Permissions(channelID, userID)
	if err != nil {
		slog.Error("failed to get permissions", "err", err, "channel", channelID, "user", userID)
		return false
	}
	return perms.Has(discord.PermissionViewChannel)
}

type memberList []discord.Member

func (ml memberList) String(i int) string {
	return ml[i].Nick + ml[i].User.DisplayName + ml[i].User.Tag()
}

func (ml memberList) Len() int { return len(ml) }

type userList []discord.User

func (ul userList) String(i int) string { return ul[i].DisplayName + ul[i].Tag() }

func (ul userList) Len() int { return len(ul) }
