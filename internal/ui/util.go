package ui

import (
	"cmp"
	"slices"
	"strings"

	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/tview"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/gdamore/tcell/v3"
)

// ConfigureBox configures the provided box according to the provided theme.
func ConfigureBox(box *tview.Box, cfg *config.Theme) *tview.Box {
	border := cfg.Border
	normalBorderStyle, activeBorderStyle := border.NormalStyle.Style, border.ActiveStyle.Style
	normalBorderSet, activeBorderSet := border.NormalSet.BorderSet, border.ActiveSet.BorderSet

	title := cfg.Title
	normalTitleStyle, activeTitleStyle := title.NormalStyle.Style, title.ActiveStyle.Style

	footer := cfg.Footer
	normalFooterStyle, activeFooterStyle := footer.NormalStyle.Style, footer.ActiveStyle.Style

	padding := border.Padding

	box.
		SetBorderStyle(normalBorderStyle).
		SetBorderSet(normalBorderSet).
		SetBorderPadding(padding[0], padding[1], padding[2], padding[3]).
		SetTitleStyle(normalTitleStyle).
		SetTitleAlignment(title.Alignment.Alignment).
		SetFooterStyle(normalFooterStyle).
		SetFooterAlignment(footer.Alignment.Alignment).
		SetBlurFunc(func() {
			box.
				SetBorderStyle(normalBorderStyle).
				SetBorderSet(normalBorderSet)
			box.SetTitleStyle(normalTitleStyle).SetFooterStyle(normalFooterStyle)
		}).
		SetFocusFunc(func() {
			box.
				SetBorderStyle(activeBorderStyle).
				SetBorderSet(activeBorderSet)
			box.SetTitleStyle(activeTitleStyle).SetFooterStyle(activeFooterStyle)
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

func ChannelToString(channel discord.Channel, icons config.Icons) string {
	var icon string
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

func MergeStyle(base, overlay tcell.Style) tcell.Style {
	fg := overlay.GetForeground()
	if fg == tcell.ColorDefault {
		fg = base.GetForeground()
	}
	bg := overlay.GetBackground()
	if bg == tcell.ColorDefault {
		bg = base.GetBackground()
	}
	style := base.Foreground(fg).Background(bg)
	style = style.Bold(base.HasBold() || overlay.HasBold())
	style = style.Dim(base.HasDim() || overlay.HasDim())
	style = style.Italic(base.HasItalic() || overlay.HasItalic())
	style = style.Blink(base.HasBlink() || overlay.HasBlink())
	style = style.Reverse(base.HasReverse() || overlay.HasReverse())
	style = style.StrikeThrough(base.HasStrikeThrough() || overlay.HasStrikeThrough())
	if base.HasUnderline() || overlay.HasUnderline() {
		style = style.Underline(true)
	}
	return style
}
