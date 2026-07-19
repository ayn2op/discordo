package mentionslist

import (
	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/discordo/internal/ui"
	"github.com/ayn2op/tview"
	"github.com/ayn2op/tview/list"
	"github.com/gdamore/tcell/v3"
	"github.com/rivo/uniseg"
)

type Model struct {
	list  *list.Model
	items []Item
}

func NewModel(cfg *config.Config) *Model {
	l := list.NewModel()
	l.Box = ui.ConfigureBox(l.Box, &cfg.Theme)
	l.
		SetSelectedStyle(tcell.StyleDefault.Reverse(true)).
		SetSnapToItems(true).
		SetTitle("Mentions")

	kbs := cfg.Keybinds.MentionsList
	l.SetKeybinds(list.Keybinds{
		SelectUp:     kbs.SelectUp.Keybind,
		SelectDown:   kbs.SelectDown.Keybind,
		SelectTop:    kbs.SelectTop.Keybind,
		SelectBottom: kbs.SelectBottom.Keybind,
	})

	b := l.BorderSet()
	b.BottomLeft, b.BottomRight = b.BottomT, b.BottomT
	l.SetBorderSet(b)

	return &Model{list: l}
}

func (m *Model) Append(item Item) {
	m.items = append(m.items, item)
}

func (m *Model) Clear() {
	m.items = nil
	m.list.Clear()
}

func (m *Model) ItemCount() int {
	return len(m.items)
}

func (m *Model) SelectedInsertText() (string, bool) {
	index := m.list.Cursor()
	if index < 0 || index >= len(m.items) {
		return "", false
	}
	return m.items[index].InsertText, true
}

func (m *Model) MaxDisplayWidth() int {
	width := 0
	for _, item := range m.items {
		width = max(width, uniseg.StringWidth(item.DisplayText))
	}
	return width
}

func (m *Model) Rebuild() {
	m.list.SetBuilder(func(index int) list.Item {
		if index < 0 || index >= len(m.items) {
			return nil
		}
		item := m.items[index]
		style := item.Style
		line := tview.NewLine(tview.NewSegment(item.DisplayText, style))
		return tview.NewTextView().
			SetScrollable(false).
			SetWrap(false).
			SetWordWrap(false).
			SetTextStyle(style).
			SetLines([]tview.Line{line})
	})

	if len(m.items) == 0 {
		m.list.SetCursor(-1)
		return
	}
	m.list.SetCursor(0)
}

var _ tview.Model = (*Model)(nil)

func (m *Model) Blur() {
	m.list.Blur()
}

func (m *Model) Focus(delegate func(tview.Model)) {
	m.list.Focus(delegate)
}

func (m *Model) HasFocus() bool {
	return m.list.HasFocus()
}

func (m *Model) Rect() (int, int, int, int) {
	return m.list.Rect()
}

func (m *Model) SetRect(x int, y int, width int, height int) {
	m.list.SetRect(x, y, width, height)
}

func (m *Model) Update(msg tview.Msg) tview.Cmd {
	return m.list.Update(msg)
}

func (m *Model) View(screen tcell.Screen) {
	m.list.View(screen)
}
