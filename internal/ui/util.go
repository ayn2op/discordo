package ui

import (
	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/tview"
	"github.com/gdamore/tcell/v2"
)

func NewConfiguredBox(box *tview.Box, cfg *config.Theme) *tview.Box {
	b := cfg.Border
	t := cfg.Title
	p := b.Padding
	box.
		SetBorderPadding(p[0], p[1], p[2], p[3]).
		SetTitleAlign(int(t.Align)).
		SetFocusFunc(func() {
			borderColor := tcell.GetColor(b.ActiveColor)
			box.SetBorderStyle(tcell.StyleDefault.Foreground(borderColor))

			titleColor := tcell.GetColor(t.ActiveColor)
			box.SetTitleStyle(tcell.StyleDefault.Foreground(titleColor))
		}).
		SetBlurFunc(func() {
			borderColor := tcell.GetColor(b.Color)
			box.SetBorderStyle(tcell.StyleDefault.Foreground(borderColor))

			titleColor := tcell.GetColor(t.Color)
			box.SetTitleStyle(tcell.StyleDefault.Foreground(titleColor))
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
