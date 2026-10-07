package ui

import (
	"cmp"
	"slices"
	"strings"

	"github.com/ayn2op/arikawa/v3/discord"
	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/ningen/v3"
	"github.com/ayn2op/tview"
	"github.com/ayn2op/tview/box"
)

// Box returns child in a box laid out and styled by cfg, highlighted while focused.
func Box(child tview.Widget, cfg *config.Config, focused bool) box.Widget {
	ui, theme := cfg.UI, cfg.Theme
	padding := ui.Border.Padding
	b := box.New(child).
		Padding(padding[0], padding[1], padding[2], padding[3]).
		TitleAlignment(ui.Title.Alignment.Alignment).
		FooterAlignment(ui.Footer.Alignment.Alignment)
	if ui.Border.Enabled {
		b = b.Borders(tview.BordersAll)
	}
	if focused {
		return b.BorderStyle(theme.Border.ActiveStyle.Style).
			BorderSet(ui.Border.ActiveSet.BorderSet).
			TitleStyle(theme.Title.ActiveStyle.Style).
			FooterStyle(theme.Footer.ActiveStyle.Style)
	}
	return b.BorderStyle(theme.Border.NormalStyle.Style).
		BorderSet(ui.Border.NormalSet.BorderSet).
		TitleStyle(theme.Title.NormalStyle.Style).
		FooterStyle(theme.Footer.NormalStyle.Style)
}

// ChannelToString returns how channel is shown in the UI: an icon for its type and its name, or the recipients' names for a DM.
func ChannelToString(channel discord.Channel, icons config.Icons, state *ningen.State) string {
	var icon string
	switch channel.Type {
	case discord.DirectMessage, discord.GroupDM:
		if channel.Name != "" {
			return channel.Name
		}

		recipients := make([]string, len(channel.DMRecipients))
		for i, r := range channel.DMRecipients {
			if state != nil && channel.Type == discord.DirectMessage {
				if rel, ok := state.RelationshipState.FullRelationship(r.ID); ok && rel.Type == discord.FriendRelationship {
					if rel.Nickname != nil && *rel.Nickname != "" {
						recipients[i] = *rel.Nickname
						continue
					}
				}
			}
			recipients[i] = r.DisplayOrUsername()
		}

		return strings.Join(recipients, ", ")

	case discord.GuildCategory:
		icon = icons.GuildCategory
	case discord.GuildText:
		icon = icons.GuildText
	case discord.GuildVoice:
		icon = icons.GuildVoice
	case discord.GuildStageVoice:
		icon = icons.GuildStageVoice

	case discord.GuildAnnouncementThread:
		icon = icons.GuildAnnouncementThread
	case discord.GuildPublicThread:
		icon = icons.GuildPublicThread
	case discord.GuildPrivateThread:
		icon = icons.GuildPrivateThread

	case discord.GuildAnnouncement:
		icon = icons.GuildAnnouncement
	case discord.GuildForum:
		icon = icons.GuildForum
	case discord.GuildStore:
		icon = icons.GuildStore
	}

	return icon + channel.Name
}

func SortGuildChannels(channels []discord.Channel) {
	slices.SortFunc(channels, func(a, b discord.Channel) int {
		return cmp.Compare(a.Position, b.Position)
	})
}

func SortPrivateChannels(channels []discord.Channel) {
	slices.SortFunc(channels, func(a, b discord.Channel) int {
		// Descending order
		return cmp.Compare(getMessageIDFromChannel(b), getMessageIDFromChannel(a))
	})
}

func getMessageIDFromChannel(channel discord.Channel) discord.MessageID {
	if channel.LastMessageID.IsValid() {
		return channel.LastMessageID
	}
	return discord.MessageID(channel.ID)
}

// IsMe reports whether id is the logged in user.
func IsMe(state *ningen.State, id discord.UserID) bool {
	me, _ := state.Cabinet.Me()
	return me != nil && id == me.ID
}
