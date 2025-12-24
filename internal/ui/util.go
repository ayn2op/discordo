package ui

import (
	"strings"

	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/tview"
	"github.com/diamondburned/arikawa/v3/discord"
)

// ConfigureBox configures the provided box according to the provided theme.
func ConfigureBox(box *tview.Box, cfg *config.Theme) *tview.Box {
	border := cfg.Border
	title := cfg.Title
	normalBorderStyle, activeBorderStyle := border.NormalStyle.Style, border.ActiveStyle.Style
	normalBorderSet, activeBorderSet := border.NormalSet.BorderSet, border.ActiveSet.BorderSet
	normalTitleStyle, activeTitleStyle := title.NormalStyle.Style, title.ActiveStyle.Style
	padding := border.Padding
	box.
		SetBorderStyle(normalBorderStyle).
		SetBorderSet(normalBorderSet).
		SetBorderPadding(padding[0], padding[1], padding[2], padding[3]).
		SetTitleStyle(normalTitleStyle).
		SetTitleAlignment(title.Alignment.Alignment).
		SetBlurFunc(func() {
			box.
				SetBorderStyle(normalBorderStyle).
				SetBorderSet(normalBorderSet)
			box.SetTitleStyle(normalTitleStyle)
		}).
		SetFocusFunc(func() {
			box.
				SetBorderStyle(activeBorderStyle).
				SetBorderSet(activeBorderSet)
			box.SetTitleStyle(activeTitleStyle)
		})

	if border.Enabled {
		box.SetBorders(tview.BordersAll)
	}

	return box
}

// Centered creates a new grid with provided primitive aligned in the center.
func Centered(p tview.Primitive, width, height int) tview.Primitive {
	return tview.NewGrid().
		SetColumns(0, width, 0).
		SetRows(0, height, 0).
		AddItem(p, 1, 1, 1, 1, 0, 0, true)
}

func ChannelToString(channel discord.Channel) string {
	switch channel.Type {
	case discord.DirectMessage, discord.GroupDM:
		if channel.Name != "" {
			return channel.Name
		}

		recipients := make([]string, len(channel.DMRecipients))
		for i, r := range channel.DMRecipients {
			recipients[i] = r.DisplayOrUsername()
		}

		return strings.Join(recipients, ", ")
	case discord.GuildText:
		return "#" + channel.Name
	case discord.GuildVoice, discord.GuildStageVoice:
		return "v-" + channel.Name
	case discord.GuildAnnouncement:
		return "a-" + channel.Name
	case discord.GuildStore:
		return "s-" + channel.Name
	case discord.GuildForum:
		return "f-" + channel.Name
	case discord.GuildPublicThread, discord.GuildPrivateThread, discord.GuildAnnouncementThread:
		return "t-" + channel.Name
	default:
		return channel.Name
	}
}
