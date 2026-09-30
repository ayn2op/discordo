// Package picker holds what discordo's pickers share.
package picker

import (
	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/discordo/internal/ui"
	"github.com/ayn2op/tview"
	"github.com/ayn2op/tview/keybind"
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
func (m Model) View(onSelect func(picker.Item) tview.Msg, onCancel tview.Msg) tview.Element {
	cfg := m.cfg
	bar := scrollbar.New().
		TrackStyle(cfg.Theme.ScrollBar.TrackStyle.Style).
		ThumbStyle(cfg.Theme.ScrollBar.ThumbStyle.Style).
		GlyphSet(cfg.Theme.ScrollBar.GlyphSet.GlyphSet).
		Arrows(scrollbar.ArrowsBoth)
	p := picker.New(m.items, &m.searchState).
		ScrollBar(bar, cfg.Theme.ScrollBar.Visibility.ScrollBarVisibility).
		Keybind(bindKeys(cfg.Keybinds.Picker)).
		ListKeybind(ui.SelectionKeybind(cfg.Keybinds.Picker.SelectionKeybinds)).
		OnChange(func(c picker.Change) tview.Msg { return changeMsg(c) }).
		OnSelect(onSelect).
		OnCancel(onCancel)
	return ui.Box(p, &cfg.Theme, true).Title(m.title)
}

func (m Model) Update(msg tview.Msg) (Model, tview.Cmd) {
	if msg, ok := msg.(changeMsg); ok {
		m.searchState.Apply(picker.Change(msg))
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

func bindKeys(kbs config.PickerKeybinds) func(tview.KeyMsg) (picker.Action, bool) {
	return func(key tview.KeyMsg) (picker.Action, bool) {
		switch {
		case keybind.Matches(key, kbs.Select.Keybind):
			return picker.ActionSelect, true
		case keybind.Matches(key, kbs.Cancel.Keybind):
			return picker.ActionCancel, true
		}
		return 0, false
	}
}
