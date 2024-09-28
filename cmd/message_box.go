package cmd

import (
	"strings"
	"github.com/rivo/tview"
	"github.com/gdamore/tcell/v2"
)

type MessageBox struct {
	*tview.TextView
}

func newMessageBox(x int, y int, w int, h int, str string, screen tcell.Screen) *MessageBox {
	mb := &MessageBox{
		TextView: tview.NewTextView(),
	}

	mb.SetDynamicColors(true)
	mb.SetRect(x, y, w, h)
	mb.SetText(str).Draw(screen)

	return mb
}

func (m *MessageBox) getLineCount() int {
	lineCount := 1
	charCount := 0

	_, _, width, _ := m.GetInnerRect()

	for _, w := range strings.Split(m.GetText(false), " ") {
		// newline char
		if strings.Index(w, "\n") != -1 {
			lineCount += 1
			charCount = len(w) + 1
			continue
		}

		charCount += len(w) + 1

		if charCount > width {
			charCount = len(w) + 1
			lineCount += 1
		}
	}

	return lineCount
}

func (m *MessageBox) Draw(screen tcell.Screen) {
	m.Box.DrawForSubclass(screen, m)
}
