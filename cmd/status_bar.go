package cmd

import (
	// "io"

	"github.com/ayn2op/tview"
	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/discordo/internal/ui"
)

type statusBar struct {
	*tview.TextView
	cfg *config.Config
}

func newStatusBar(cfg *config.Config) *statusBar {
	sb := &statusBar{
		TextView: tview.NewTextView(),
		cfg:      cfg,
	}

	sb.Box = ui.ConfigureBox(sb.Box, &cfg.Theme)
	sb.Box.
		SetBorders(tview.BordersNone)
	sb.
		SetRegions(true).
		SetWrap(false).
		// SetRoot(tview.NewTreeNode("")).
		// SetTopLevel(1).
		// SetGraphics(cfg.Theme.GuildsTree.Graphics).
		// SetGraphicsColor(tcell.GetColor(cfg.Theme.GuildsTree.GraphicsColor)).
		//SetSelectedFunc(sb.onSelected).
		SetTitle("Status Bar!")
		//SetInputCapture(sb.onInputCapture)

	return sb
}

func (sb *statusBar) setText(t string) {


	sb.SetTitle(t)
}