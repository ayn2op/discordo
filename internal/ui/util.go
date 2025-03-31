package ui

import (
	"github.com/ayn2op/discordo/internal/config"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func NewConfiguredBox(box *tview.Box, cfg *config.Theme) *tview.Box {
	b := cfg.Border
	p := b.Padding
	box.
		SetBorder(cfg.Border.Enabled).
		SetBorderPadding(p[0], p[1], p[2], p[3]).
		SetTitleAlign(tview.AlignLeft).
		SetFocusFunc(func() {
			box.SetBorderColor(tcell.GetColor(b.ActiveColor))
			box.SetTitleColor(tcell.GetColor(cfg.ActiveTitleColor))
		}).
		SetBlurFunc(func() {
			box.SetBorderColor(tcell.GetColor(b.Color))
			box.SetTitleColor(tcell.GetColor(cfg.TitleColor))
		})
	return box
}

func Centered(p tview.Primitive, width, height int) tview.Primitive {
	return tview.NewGrid().
		SetColumns(0, width, 0).
		SetRows(0, height, 0).
		AddItem(p, 1, 1, 1, 1, 0, 0, true)
}
