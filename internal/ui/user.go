package ui

import (
	"github.com/ayn2op/arikawa/v3/discord"
	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/ningen/v3"
	"github.com/ayn2op/tview"
)

// IsMe reports whether id is the logged in user.
func IsMe(state *ningen.State, id discord.UserID) bool {
	me, _ := state.Cabinet.Me()
	return me != nil && id == me.ID
}

// StatusStyle returns the style of a user with status: success while online, warning while idle, danger while not to be disturbed, and secondary otherwise.
func StatusStyle(palette config.ThemeConfig, status discord.Status) tview.Style {
	switch status {
	case discord.OnlineStatus:
		return Foreground(palette.Success.Style)
	case discord.IdleStatus:
		return Foreground(palette.Warning.Style)
	case discord.DoNotDisturbStatus:
		return Foreground(palette.Danger.Style)
	}
	return Foreground(palette.Secondary.Base.Style)
}
