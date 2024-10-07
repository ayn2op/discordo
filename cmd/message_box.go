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
	mb.SetRegions(true)
	mb.SetBackgroundColor(tcell.GetColor(cfg.Theme.BackgroundColor))

	return mb
}

func (m *MessageBox) getLineCount(width int) int {
	lineCount := 0
	charCount := 0

	for _, w := range strings.Split(m.Content, " ") {
		// Don't count \n, since GetOriginalLineCount() already does that
		if strings.Index(w, "\n") != -1 {
			charCount = 0
			continue
		}

		charCount += len(w) + 1
		for charCount > width  {
			lineCount++
			charCount -= width
		}
	}

	return lineCount + m.GetOriginalLineCount()
}

func (m *MessageBox) Draw(screen tcell.Screen) {
	m.DrawForSubclass(screen, m)
}

// To render the message, Draw() needs to be called once after any TextView func that returns itself
// There has to be a better way of handling that
func (m *MessageBox) Render(highlight bool, screen tcell.Screen) {
	if highlight {
		m.Highlight("msg").Draw(screen)
	} else {
		m.Highlight().Draw(screen)
	}
}
