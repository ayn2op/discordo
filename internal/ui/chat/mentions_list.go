package chat

import (
	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/discordo/internal/ui"
	"github.com/ayn2op/tview"
	"github.com/ayn2op/tview/list"
	"github.com/gdamore/tcell/v3"
)

type mentionsListItem struct {
	insertText  string
	displayText string
	style       tcell.Style
}

type mentionsList struct {
	*list.Model
	items []mentionsListItem
}

func newMentionsList(cfg *config.Config) *mentionsList {
	m := &mentionsList{
		Model: list.NewModel(),
	}
	m.SetKeybinds(list.Keybinds{
		SelectUp:     cfg.Keybinds.MentionsList.Up.Keybind,
		SelectDown:   cfg.Keybinds.MentionsList.Down.Keybind,
		SelectTop:    cfg.Keybinds.MentionsList.Top.Keybind,
		SelectBottom: cfg.Keybinds.MentionsList.Bottom.Keybind,
	})

	m.Box = ui.ConfigureBox(m.Box, &cfg.Theme)
	m.SetSnapToItems(true).SetTitle("Mentions")

	b := m.GetBorderSet()
	b.BottomLeft, b.BottomRight = b.BottomT, b.BottomT
	m.SetBorderSet(b)

	return m
}

func (m *mentionsList) append(item mentionsListItem) {
	m.items = append(m.items, item)
}

func (m *mentionsList) clear() {
	m.items = nil
	m.Clear()
}

func (m *mentionsList) rebuild() {
	m.SetBuilder(func(index int, cursor int) list.Item {
		if index < 0 || index >= len(m.items) {
			return nil
		}

		item := m.items[index]
		style := item.style
		if index == cursor {
			style = style.Reverse(true)
		}
		line := tview.NewLine(tview.NewSegment(item.displayText, style))

		return tview.NewTextView().
			SetScrollable(false).
			SetWrap(false).
			SetWordWrap(false).
			SetTextStyle(style).
			SetLines([]tview.Line{line})
	})

	if len(m.items) == 0 {
		m.SetCursor(-1)
		return
	}
	m.SetCursor(0)
}

func (m *mentionsList) itemCount() int {
	return len(m.items)
}

func (m *mentionsList) selectedInsertText() (string, bool) {
	index := m.Cursor()
	if index < 0 || index >= len(m.items) {
		return "", false
	}
	return m.items[index].insertText, true
}

func (m *mentionsList) maxDisplayWidth() int {
	width := 0
	for _, item := range m.items {
		width = max(width, tview.TaggedStringWidth(item.displayText))
	}
	return width
}
