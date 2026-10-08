package mentionslist

import (
	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/discordo/internal/ui"
	"github.com/ayn2op/tview"
	"github.com/ayn2op/tview/layout"
	"github.com/ayn2op/tview/list"
	"github.com/ayn2op/tview/scrollbar"
	"github.com/ayn2op/tview/viewport"
	"github.com/rivo/uniseg"
)

// Model shows the mention suggestions as a list in a box.
type Model struct {
	cfg            *config.Config
	selectionState list.SelectionState
	scrollState    viewport.ScrollState
	items          []Item
}

func NewModel(cfg *config.Config) Model {
	return Model{cfg: cfg, selectionState: list.NewSelectionState()}
}

// View shows the list in a box whose bottom corners join the composer's border below it.
func (m Model) View() tview.Widget {
	set := m.cfg.UI.Border.NormalSet.BorderSet
	set.BottomLeft, set.BottomRight = set.BottomT, set.BottomT
	return ui.Box(m.listView(), m.cfg, false).Title("Mentions").BorderSet(set)
}

func (Model) Init() tview.Cmd { return nil }

func (m Model) listView() viewport.Widget {
	l := list.New(m.selectionState, len(m.items), func(i int) tview.Widget {
		item := m.items[i]
		return row{text: item.DisplayText, style: item.Style}
	}).
		SelectedStyle(m.cfg.Theme.MentionsList.SelectedStyle.Style).
		Keybind(ui.SelectionKeybind(m.cfg.Keybinds.MentionsList.SelectionKeybinds)).
		// The list is only shown while mentions are being completed, so it takes its keys first.
		Focused(true).
		OnChange(func(a list.Change) tview.Msg { return listMsg(a) })
	return viewport.New(l, m.scrollState).
		Target(l.Target).
		ScrollBar(scrollbar.New(), viewport.ScrollBarVisibilityAutomatic).
		OnChange(func(a viewport.Change) tview.Msg { return scrollMsg(a) })
}

func (m Model) Update(msg tview.Msg) (Model, tview.Cmd) {
	switch msg := msg.(type) {
	case listMsg:
		m.selectionState.Apply(list.Change(msg))
		m.scrollState.ScrollToTarget()
	case scrollMsg:
		m.scrollState.Apply(viewport.Change(msg))
	}
	return m, nil
}

func (m *Model) Append(item Item) {
	m.items = append(m.items, item)
}

func (m *Model) Clear() {
	m.items = nil
	m.selectionState.SetCursor(-1)
}

func (m Model) ItemCount() int {
	return len(m.items)
}

func (m Model) SelectedInsertText() (string, bool) {
	index := m.selectionState.Cursor()
	if index < 0 || index >= len(m.items) {
		return "", false
	}
	return m.items[index].InsertText, true
}

func (m Model) MaxDisplayWidth() int {
	width := 0
	for _, item := range m.items {
		width = max(width, uniseg.StringWidth(item.DisplayText))
	}
	return width
}

func (m *Model) Rebuild() {
	m.selectionState.SetCursor(min(0, len(m.items)-1))
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
