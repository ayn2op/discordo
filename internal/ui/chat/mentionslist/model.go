package mentionslist

import (
	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/discordo/internal/ui"
	"github.com/ayn2op/tview"
	"github.com/ayn2op/tview/list"
	"github.com/gdamore/tcell/v3"
	"github.com/rivo/uniseg"
)

// Model shows the mention suggestions as a list in a box.
type Model struct {
	cfg            *config.Config
	selectionState list.SelectionState
	items          []Item
	entries        []list.Item
	keybinds       list.Keybinds
}

// Msg moves the list.
type Msg list.Action

func NewModel(cfg *config.Config) Model {
	m := Model{cfg: cfg, selectionState: list.NewSelectionState()}

	kbs := cfg.Keybinds.MentionsList
	m.keybinds = list.Keybinds{
		SelectUp:     kbs.SelectUp.Keybind,
		SelectDown:   kbs.SelectDown.Keybind,
		SelectTop:    kbs.SelectTop.Keybind,
		SelectBottom: kbs.SelectBottom.Keybind,
	}
	return m
}

// View shows the list in a box whose bottom corners join the composer's border below it.
func (m Model) View() tview.Element {
	set := m.cfg.Theme.Border.NormalSet.BorderSet
	set.BottomLeft, set.BottomRight = set.BottomT, set.BottomT
	return ui.Box(m.listView(), &m.cfg.Theme, false).Title("Mentions").BorderSet(set)
}

func (Model) Init() tview.Cmd { return nil }

func (m Model) listView() list.Widget {
	return list.New(&m.selectionState, len(m.entries), func(i int) list.Item { return m.entries[i] }).
		SelectedStyle(tcell.StyleDefault.Reverse(true)).
		Keybinds(m.keybinds).
		// The list is only shown while mentions are being completed, so it takes its keys first.
		Focused(true).
		OnAction(func(a list.Action) tview.Msg { return Msg(a) })
}

func (m Model) Update(msg tview.Msg) (Model, tview.Cmd) {
	if msg, ok := msg.(Msg); ok {
		m.selectionState.Perform(list.Action(msg))
	}
	return m, nil
}

func (m *Model) Append(item Item) {
	m.items = append(m.items, item)
}

func (m *Model) Clear() {
	m.items, m.entries = nil, nil
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
	m.entries = make([]list.Item, len(m.items))
	for i, item := range m.items {
		m.entries[i] = row{text: item.DisplayText, style: item.Style}
	}
	m.selectionState.SetCursor(min(0, len(m.items)-1))
}

var _ tview.Model[Model] = Model{}

// row is a mention on one line in its style.
type row struct {
	text  string
	style tcell.Style
}

func (row) Rows(int) int { return 1 }

func (r row) Draw(screen tview.Screen, area tview.Rectangle) {
	for x := area.X; x < area.X+area.Width; x++ {
		screen.Put(x, area.Y, " ", r.style)
	}
	tview.Print(screen, r.text, area.X, area.Y, area.Width, tview.AlignmentLeft, r.style)
}

func (row) Handle(msg tview.Msg, area tview.Rectangle) tview.Msg { return msg }
