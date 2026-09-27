package attachmentspicker

import (
	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/discordo/internal/ui"
	"github.com/ayn2op/tview"
	"github.com/ayn2op/tview/picker"
)

type Item struct {
	Label  string
	Action tview.Cmd
}

type Model struct {
	items       picker.Items
	searchState picker.SearchState
	cfg         *config.Config
}

func NewModel(cfg *config.Config) Model {
	return Model{searchState: picker.NewSearchState(), cfg: cfg}
}

var _ tview.Model[Model] = Model{}

// actionMsg changes the picker.
type actionMsg picker.Action

func (Model) Init() tview.Cmd { return nil }

// View shows the picker in a box titled Attachments.
func (m Model) View() tview.Element {
	p := ui.Picker(m.items, &m.searchState, m.cfg).
		OnAction(func(a picker.Action) tview.Msg { return actionMsg(a) }).
		OnSelect(func(item picker.Item) tview.Msg {
			action, ok := item.Reference.(tview.Cmd)
			if !ok {
				return nil
			}
			return SelectedMsg{Action: action}
		}).
		OnCancel(CancelMsg{})
	return ui.Box(p, &m.cfg.Theme, true).Title("Attachments")
}

func (m *Model) SetItems(items []Item) {
	pickerItems := make(picker.Items, len(items))
	for i, item := range items {
		pickerItems[i] = picker.Item{Text: item.Label, Reference: item.Action}
	}
	m.items = pickerItems
	m.searchState.Reset()
}

func (m Model) Update(msg tview.Msg) (Model, tview.Cmd) {
	if msg, ok := msg.(actionMsg); ok {
		m.searchState.Perform(picker.Action(msg))
	}
	return m, nil
}
