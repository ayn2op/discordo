package attachmentspicker

import (
	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/discordo/internal/ui"
	"github.com/ayn2op/tview"
	"github.com/ayn2op/tview/picker"
)

type Item struct {
	Label string
	Open  tview.Cmd
}

type Model struct{ *picker.Model }

func NewModel(cfg *config.Config) *Model {
	m := &Model{Model: picker.NewModel()}
	ui.ConfigurePicker(m.Model, cfg, "Attachments")
	return m
}

func (m *Model) SetItems(items []Item) {
	pickerItems := make(picker.Items, len(items))
	for i, item := range items {
		pickerItems[i] = picker.Item{Text: item.Label, Reference: item.Open}
	}
	m.Model.SetItems(pickerItems)
}

func (m *Model) Update(msg tview.Msg) tview.Cmd {
	switch msg := msg.(type) {
	case picker.SelectedMsg:
		open, ok := msg.Reference.(tview.Cmd)
		if !ok {
			return nil
		}
		return func() tview.Msg { return SelectedMsg{Open: open} }
	case picker.CancelMsg:
		return func() tview.Msg { return CancelMsg{} }
	}

	return m.Model.Update(msg)
}
