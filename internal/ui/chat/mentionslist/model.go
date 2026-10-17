package mentionslist

import (
	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/discordo/internal/ui"
	"github.com/ayn2op/tview"
	"github.com/ayn2op/tview/layout"
	"github.com/ayn2op/tview/list"
	"github.com/ayn2op/tview/scrollbar"
	"github.com/ayn2op/tview/viewport"
	"github.com/rockorager/go-uucode"
)

// Model shows the mention suggestions as a list in a box.
type Model struct {
	cfg *config.Config

	items []Item
	// cursor is the selected item, or -1 for none.
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

	return ui.Box(list, m.cfg, false).Title("Mentions").BorderSet(set).Width(width).Height(layout.Shrink)
}

func (Model) Init() tview.Cmd { return nil }

func (m Model) listView() viewport.Widget {
	l := list.New(m.cursor, len(m.items), func(i int) tview.Widget {
		item := m.items[i]
		return row{text: item.DisplayText, style: item.Style}
	}).
		SelectedStyle(m.cfg.Theme.MentionsList.SelectedStyle.Style).
		Keybind(ui.SelectionKeybind(m.cfg.Keybinds.MentionsList.SelectionKeybinds)).
		// The list is only shown while mentions are being completed, so it takes its keys first.
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

// row is a mention on one line in its style.
type row struct {
	text  string
	style tview.Style
}

// Size returns Fill and a height of one row.
func (row) Size() (width, height layout.Length) { return layout.Fill, layout.Fixed(1) }

// Layout returns a height of one row.
func (row) Layout(limits layout.Limits) layout.Size {
	return layout.Atomic(limits, layout.Fill, layout.Fixed(1))
}

func (r row) Draw(screen tview.Screen, area tview.Rectangle) {
	for x := area.X; x < area.X+area.Width; x++ {
		screen.Put(x, area.Y, " ", r.style)
	}
	tview.Print(screen, r.text, area.X, area.Y, area.Width, tview.AlignmentLeft, r.style)
}

func (row) Handle(msg tview.Msg, area tview.Rectangle) tview.Msg { return msg }
