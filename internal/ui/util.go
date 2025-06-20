package ui

import (
	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/tview"
)

// ConfigureBox configures the provided box according to the provided theme.
func ConfigureBox(box *tview.Box, cfg *config.Theme) *tview.Box {
	border := cfg.Border
	title := cfg.Title
	normalBorderStyle, activeBorderStyle := border.NormalStyle.Style, border.ActiveStyle.Style
	normalTitleStyle, activeTitleStyle := title.NormalStyle.Style, title.ActiveStyle.Style
	p := border.Padding
	box.
		SetBorderStyle(normalBorderStyle).
		SetBorderSet(border.Set.BorderSet).
		SetBorderPadding(p[0], p[1], p[2], p[3]).
		SetTitleStyle(normalTitleStyle).
		SetTitleAlignment(title.Alignment.Alignment).
		SetFocusFunc(func() {
			box.SetBorderStyle(activeBorderStyle)
			box.SetTitleStyle(activeTitleStyle)
		}).
		SetBlurFunc(func() {
			box.SetBorderStyle(normalBorderStyle)
			box.SetTitleStyle(normalTitleStyle)
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
