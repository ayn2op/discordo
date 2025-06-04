package ui

import (
	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/tview"
)

func NewConfiguredBox(box *tview.Box, cfg *config.Theme) *tview.Box {
	b := cfg.Border
	t := cfg.Title
	p := b.Padding
	box.
		SetBorderStyle(b.Style.Style).
		SetBorderSet(b.Set.BorderSet).
		SetBorderPadding(p[0], p[1], p[2], p[3]).
		SetTitleStyle(t.Style.Style).
		SetTitleAlignment(t.Alignment.Alignment).
		SetFocusFunc(func() {
			box.SetBorderStyle(b.ActiveStyle.Style)
			box.SetTitleStyle(t.ActiveStyle.Style)
		}).
		SetBlurFunc(func() {
			box.SetBorderStyle(b.Style.Style)
			box.SetTitleStyle(t.Style.Style)
		})

	if b.Enabled {
		box.SetBorders(tview.BordersAll)
	}

	return box
}

func Centered(p tview.Primitive, width, height int) tview.Primitive {
	return tview.NewGrid().
		SetColumns(0, width, 0).
		SetRows(0, height, 0).
		AddItem(p, 1, 1, 1, 1, 0, 0, true)
}
