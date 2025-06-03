package ui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"tui/internal/config"
)

func NewConfiguredBox(box *tview.Box, cfg *config.Theme) *tview.Box {
	b := cfg.Border
	t := cfg.Title
	p := b.Padding
	box.
		SetBorder(cfg.Border.Enabled).
		SetBorderColor(tcell.GetColor(b.Color)).
		SetBorderPadding(p[0], p[1], p[2], p[3]).
		SetTitleAlign(int(t.Align)).
		SetFocusFunc(func() {
			box.SetBorderColor(tcell.GetColor(b.ActiveColor))
			box.SetTitleColor(tcell.GetColor(t.ActiveColor))
		}).
		SetBlurFunc(func() {
			box.SetBorderColor(tcell.GetColor(b.Color))
			box.SetTitleColor(tcell.GetColor(t.Color))
		})
	return box
}

func Centered(p tview.Primitive, width, height int) tview.Primitive {
	return tview.NewGrid().
		SetColumns(0, width, 0).
		SetRows(0, height, 0).
		AddItem(p, 1, 1, 1, 1, 0, 0, true)
}
