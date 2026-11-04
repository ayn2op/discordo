// Package themepicker picks the theme, previewing the selected one.
package themepicker

import (
	"slices"

	"github.com/ayn2op/discordo/internal/config"
	uipicker "github.com/ayn2op/discordo/internal/ui/picker"
	"github.com/ayn2op/tview"
	"github.com/ayn2op/tview/cmd"
	"github.com/ayn2op/tview/picker"
	"github.com/ayn2op/tview/text"
)

type Model struct {
	uipicker.Model
	// previous is the theme the picker was shown with, and previewed the last one previewed.
	previous, previewed string
}

func NewModel(cfg *config.Config) Model {
	return Model{Model: uipicker.NewModel(cfg, "Themes")}
}

// Previous returns the theme the picker was shown with.
func (m Model) Previous() string {
	return m.previous
}

// Previewed returns the theme previewed last.
func (m Model) Previewed() string {
	return m.previewed
}

var _ tview.Model[Model] = Model{}

func (m Model) Update(msg tview.Msg) (Model, tview.Cmd) {
	switch msg := msg.(type) {
	case ShowMsg:
		names := config.Themes()
		items := make(picker.Items, len(names))
		for i, name := range names {
			items[i] = picker.Item{Content: text.New(name).Wrapping(text.WrappingNone), FilterText: name, Reference: name}
		}
		m.SetItems(items)
		m.Select(slices.Index(names, msg.Theme))
		m.previous, m.previewed = msg.Theme, msg.Theme
		return m, nil
	}

	var c tview.Cmd
	m.Model, c = m.Model.Update(msg)
	if item, ok := m.Selected(); ok && item.Reference != m.previewed {
		m.previewed = item.Reference.(string)
		preview := PreviewMsg{Theme: m.previewed}
		c = cmd.Batch{c, cmd.Task(func() tview.Msg { return preview })}
	}
	return m, c
}

// View shows the picker in a box titled Themes.
func (m Model) View() tview.Widget {
	return m.Model.View(func(item picker.Item) tview.Msg {
		theme, ok := item.Reference.(string)
		if !ok {
			return nil
		}
		return SelectedMsg{Theme: theme}
	}, CancelMsg{})
}
