package attachmentspicker

import (
	"github.com/ayn2op/discordo/internal/config"
	uipicker "github.com/ayn2op/discordo/internal/ui/picker"
	"github.com/ayn2op/tview"
	"github.com/ayn2op/tview/picker"
)

type Model struct {
	uipicker.Model
}

func NewModel(cfg *config.Config) Model {
	return Model{uipicker.NewModel(cfg, "Attachments")}
}

var _ tview.Model[Model] = Model{}

// View shows the picker in a box titled Attachments.
func (m Model) View() tview.Element {
	return m.Model.View(func(item picker.Item) tview.Msg {
		action, ok := item.Reference.(tview.Cmd)
		if !ok {
			return nil
		}
		return SelectedMsg{Action: action}
	}, CancelMsg{})
}

func (m Model) Update(msg tview.Msg) (Model, tview.Cmd) {
	var cmd tview.Cmd
	m.Model, cmd = m.Model.Update(msg)
	return m, cmd
}
