package commandslist

import (
	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/discordo/internal/ui"
	"github.com/ayn2op/tview"
	"github.com/ayn2op/tview/layout"
	"github.com/ayn2op/tview/list"
	"github.com/ayn2op/tview/scrollbar"
	"github.com/ayn2op/tview/text"
	"github.com/ayn2op/tview/viewport"
	"github.com/rockorager/go-uucode"
)

// Model shows command suggestions as a list in a box.
type Model struct {
	cfg *config.Config

	items       []Item
	cursor      int
	scrollState viewport.ScrollState
}

func NewModel(cfg *config.Config) Model {
	return Model{cfg: cfg, cursor: -1}
}

// View shows the list in a box whose bottom corners join the composer's border below it.
func (m Model) View() tview.Widget {
	set := m.cfg.UI.Border.NormalSet.BorderSet
	set.BottomLeft, set.BottomRight = set.BottomT, set.BottomT
	cfg := m.cfg.UI.MentionsList
	list, width := m.listView().Height(layout.Shrink.Max(int(cfg.MaxHeight))), layout.Fill
	if cfg.MinWidth != 0 {
		list, width = list.Width(layout.Fixed(max(int(cfg.MinWidth), m.maxDisplayWidth()))), layout.Shrink
	}

	return ui.Box(list, m.cfg, false).Title(ui.Label("Commands", m.cfg.Theme.Title, false)).BorderSet(set).Width(width).Height(layout.Shrink)
}

func (Model) Init() tview.Cmd { return nil }

func (m Model) listView() viewport.Widget {
	l := list.New(m.cursor, len(m.items), func(i int) tview.Widget {
		item := m.items[i]
		return text.New(item.DisplayText).Style(item.Style).Wrapping(text.WrappingNone)
	}).
		SelectedStyle(m.cfg.Theme.MentionsList.SelectedStyle.Style).
		Keybind(ui.SelectionKeybind(m.cfg.Keybinds.MentionsList.SelectionKeybinds)).
		Focused(true).
		OnChange(func(cursor int) tview.Msg { return listMsg(cursor) })
	return viewport.New(l, m.scrollState).
		Target(l.Target(m.cursor)).
		ScrollBar(scrollbar.New(), viewport.ScrollBarVisibilityAutomatic).
		OnChange(func(s *viewport.ScrollState) tview.Msg { return scrollMsg(*s) })
}

func (m Model) Update(msg tview.Msg) (Model, tview.Cmd) {
	switch msg := msg.(type) {
	case listMsg:
		m.cursor = int(msg)
		m.scrollState.ScrollToTarget()
	case scrollMsg:
		m.scrollState = viewport.ScrollState(msg)
	}
	return m, nil
}

func (m *Model) Append(item Item) {
	m.items = append(m.items, item)
}

func (m *Model) Clear() {
	m.items = nil
	m.cursor = -1
}

func (m Model) ItemCount() int {
	return len(m.items)
}

func (m Model) SelectedInsertText() (string, bool) {
	index := m.cursor
	if index < 0 || index >= len(m.items) {
		return "", false
	}
	return m.items[index].InsertText, true
}

func (m Model) maxDisplayWidth() int {
	width := 0
	for _, item := range m.items {
		width = max(width, uucode.StringWidth(item.DisplayText))
	}
	return width
}

func (m *Model) Rebuild() {
	m.cursor = min(0, len(m.items)-1)
	m.scrollState.ScrollToTarget()
}

var _ tview.Model[Model] = Model{}
