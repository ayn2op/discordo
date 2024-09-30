package cmd

import (
	"strings"
	"github.com/rivo/tview"
	"github.com/gdamore/tcell/v2"
	"github.com/diamondburned/arikawa/v3/discord"
)

type MessageBox struct {
	*tview.TextView
	*discord.Message
}

func newMessageBox() *MessageBox {
	mb := &MessageBox{
		TextView: tview.NewTextView(),
	}

	mb.SetDynamicColors(true)
	mb.SetWordWrap(true)
	mb.SetRegions(true)
	mb.SetBackgroundColor(tcell.GetColor(cfg.Theme.BackgroundColor))

	return mb
}

func (m *MessageBox) getLineCount() int {
	lineCount := 0
	charCount := 0

	_, _, width, _ := m.GetInnerRect()

	for _, w := range strings.Split(m.Content, " ") {
		// don't count \n, since GetOriginalLineCount() already does that
		if strings.Index(w, "\n") != -1 {
			charCount = 0
			continue
		}
		charCount += len(w) + 1
		if charCount > width  {
			lineCount++
			charCount = len(w) + 1
		}
	}

	return lineCount + m.GetOriginalLineCount()
}

func (m *MessageBox) Draw(screen tcell.Screen) {
	m.DrawForSubclass(screen, m)
}
