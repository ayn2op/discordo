// Package autocomplete shows suggestions above the composer.
package autocomplete

import (
	"maps"

	"github.com/ayn2op/arikawa/v3/discord"
	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/discordo/internal/ui"
	"github.com/ayn2op/tview"
	tviewimage "github.com/ayn2op/tview/image"
	"github.com/ayn2op/tview/layout"
	"github.com/ayn2op/tview/list"
	"github.com/ayn2op/tview/scrollbar"
	"github.com/ayn2op/tview/text"
	"github.com/ayn2op/tview/viewport"
	"github.com/rockorager/go-uucode"
)

// Model is a list of suggestions, hidden while it has none.
type Model struct {
	cfg   *config.Config
	title string
	items []Item
	// images are blank while they load.
	images      map[discord.URL]tviewimage.Widget
	cursor      int
	scrollState viewport.ScrollState
}

func NewModel(cfg *config.Config) Model {
	return Model{cfg: cfg}
}

// Visible reports whether there are suggestions.
func (m Model) Visible() bool {
	return len(m.items) != 0
}

// Selected returns the selected item, or false if hidden.
func (m Model) Selected() (Item, bool) {
	if !m.Visible() {
		return nil, false
	}
	return m.items[m.cursor], true
}

func (m Model) listView() viewport.Widget {
	l := list.New(m.cursor, len(m.items), func(i int) tview.Widget {
		return text.Rich(m.items[i].Line(m.images)...).Wrapping(text.WrappingNone)
	}).
		SelectedStyle(m.cfg.Theme.Autocomplete.SelectedStyle.Style).
		Keybind(ui.SelectionKeybind(m.cfg.Keybinds.Autocomplete.SelectionKeybinds)).
		// Shown only while completing, so it takes keys first.
		Focused(true).
		OnChange(func(cursor int) tview.Msg { return listMsg(cursor) })
	return viewport.New(l, m.scrollState).
		Target(l.Target(m.cursor)).
		ScrollBar(scrollbar.New(), viewport.ScrollBarVisibilityAutomatic).
		OnChange(func(s *viewport.ScrollState) tview.Msg { return scrollMsg(*s) })
}

func (m Model) maxDisplayWidth() int {
	width := 0
	for _, item := range m.items {
		lineWidth := 0
		for _, span := range item.Line(m.images) {
			lineWidth += uucode.StringWidth(span.Text)
		}
		width = max(width, lineWidth)
	}
	return width
}

var _ tview.Model[Model] = Model{}

func (m Model) Init() tview.Cmd { return nil }

func (m Model) Update(msg tview.Msg) (Model, tview.Cmd) {
	switch msg := msg.(type) {
	case SetItemsMsg:
		m.title, m.items, m.images, m.cursor = msg.Title, msg.Items, map[discord.URL]tviewimage.Widget{}, 0
		m.scrollState.ScrollToTarget()
	case listMsg:
		m.cursor = int(msg)
		m.scrollState.ScrollToTarget()
	case scrollMsg:
		m.scrollState = viewport.ScrollState(msg)
	case ui.ImagesUpdatedMsg:
		// Images arrive for other panes too.
		if m.Visible() {
			maps.Copy(m.images, msg)
		}
	}
	return m, nil
}

// View joins the box's bottom corners to the composer's border.
func (m Model) View() tview.Widget {
	set := m.cfg.UI.Border.NormalSet.BorderSet
	set.BottomLeft, set.BottomRight = set.BottomT, set.BottomT
	cfg := m.cfg.UI.Autocomplete
	list, width := m.listView().Height(layout.Shrink.Max(int(cfg.MaxHeight))), layout.Fill
	if cfg.MinWidth != 0 {
		list, width = list.Width(layout.Fixed(max(int(cfg.MinWidth), m.maxDisplayWidth()))), layout.Shrink
	}

	return ui.Box(list, m.cfg, false).Title(ui.Label(m.title, m.cfg.Theme.Title, false)).BorderSet(set).Width(width).Height(layout.Shrink)
}
