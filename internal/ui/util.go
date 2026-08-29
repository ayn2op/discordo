package ui

import (
	"cmp"
	"slices"
	"strings"

	"github.com/ayn2op/arikawa/v3/discord"
	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/ningen/v3"
	"github.com/ayn2op/tview"
	"github.com/ayn2op/tview/grid"
	"github.com/ayn2op/tview/picker"
)

// ConfigureBox configures the provided box according to the provided theme.
func ConfigureBox(box *tview.Box, cfg *config.Theme) {
	padding := cfg.Border.Padding
	BlurBox(box, cfg)
	box.
		SetBorderPadding(padding[0], padding[1], padding[2], padding[3]).
		SetTitleAlignment(cfg.Title.Alignment.Alignment).
		SetFooterAlignment(cfg.Footer.Alignment.Alignment)

	if cfg.Border.Enabled {
		box.SetBorders(tview.BordersAll)
	}
}

func FocusBox(box *tview.Box, cfg *config.Theme) {
	box.SetBorderStyle(cfg.Border.ActiveStyle.Style).
		SetBorderSet(cfg.Border.ActiveSet.BorderSet).
		SetTitleStyle(cfg.Title.ActiveStyle.Style).
		SetFooterStyle(cfg.Footer.ActiveStyle.Style)
}

func BlurBox(box *tview.Box, cfg *config.Theme) {
	box.SetBorderStyle(cfg.Border.NormalStyle.Style).
		SetBorderSet(cfg.Border.NormalSet.BorderSet).
		SetTitleStyle(cfg.Title.NormalStyle.Style).
		SetFooterStyle(cfg.Footer.NormalStyle.Style)
}

func ConfigurePicker(model *picker.Model, cfg *config.Config, title string) {
	model.Box = tview.NewBox()
	ConfigureBox(model.Box, &cfg.Theme)
	FocusBox(model.Box, &cfg.Theme)

	model.SetTitle(title)
	model.SetScrollBarVisibility(cfg.Theme.ScrollBar.Visibility.ScrollBarVisibility)
	model.SetScrollBar(tview.NewScrollBar().
		SetTrackStyle(cfg.Theme.ScrollBar.TrackStyle.Style).
		SetThumbStyle(cfg.Theme.ScrollBar.ThumbStyle.Style).
		SetGlyphSet(cfg.Theme.ScrollBar.GlyphSet.GlyphSet))
	model.SetKeybinds(picker.Keybinds{
		Cancel:       cfg.Keybinds.Picker.Cancel.Keybind,
		SelectUp:     cfg.Keybinds.Picker.SelectUp.Keybind,
		SelectDown:   cfg.Keybinds.Picker.SelectDown.Keybind,
		SelectTop:    cfg.Keybinds.Picker.SelectTop.Keybind,
		SelectBottom: cfg.Keybinds.Picker.SelectBottom.Keybind,
		Select:       cfg.Keybinds.Picker.Select.Keybind,
	})
}

// Centered creates a new grid with provided primitive aligned in the center.
func Centered(m tview.Model, width, height int) tview.Model {
	return grid.NewModel().
		SetColumns(0, width, 0).
		SetRows(0, height, 0).
		AddItem(m, 1, 1, 1, 1, 0, 0, true)
}

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
