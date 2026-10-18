// Package picker holds what discordo's pickers share.
package picker

import (
	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/discordo/internal/ui"
	"github.com/ayn2op/tview"
	"github.com/ayn2op/tview/picker"
	"github.com/ayn2op/tview/scrollbar"
)

// Model is a picker of items shown in a box titled title.
type Model struct {
	items       picker.Items
	searchState picker.SearchState
	cfg         *config.Config
	title       string
}

func NewModel(cfg *config.Config, title string) Model {
	return Model{searchState: picker.NewSearchState(), cfg: cfg, title: title}
}

func (Model) Init() tview.Cmd { return nil }

// View shows the picker, turning the selected item into the message onSelect returns and canceling with onCancel.
func (m Model) View(onSelect func(picker.Item) tview.Msg, onCancel tview.Msg) tview.Widget {
	cfg := m.cfg
	bar := scrollbar.New().
		SymbolSet(cfg.UI.ScrollBar.SymbolSet.SymbolSet).
		Style(cfg.Theme.ScrollBar.TrackStyle.Style).
		ThumbStyle(cfg.Theme.ScrollBar.ThumbStyle.Style)
	p := picker.New(m.items, &m.searchState).
		ScrollBar(bar, cfg.UI.ScrollBar.Visibility.ScrollBarVisibility).
		Keybind(pickerKeybind(cfg.Keybinds.Picker)).
		ListKeybind(ui.SelectionKeybind(cfg.Keybinds.Picker.SelectionKeybinds)).
		OnChange(func(s *picker.SearchState) tview.Msg { return changeMsg(*s) }).
		OnSelect(onSelect).
		OnCancel(onCancel)
	return ui.Box(p, cfg, true).Title(ui.Label(m.title, cfg.Theme.Title, true))
}

func (m Model) Update(msg tview.Msg) (Model, tview.Cmd) {
	if msg, ok := msg.(changeMsg); ok {
		m.searchState = picker.SearchState(msg)
	}
	return m, nil
}

// SetItems replaces the items and clears the query.
func (m *Model) SetItems(items picker.Items) {
	m.items = items
	m.searchState.Reset()
}

// Reset clears the query.
func (m *Model) Reset() { m.searchState.Reset() }
