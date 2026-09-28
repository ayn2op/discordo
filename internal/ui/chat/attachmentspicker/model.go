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

// changeMsg changes the picker.
type changeMsg picker.Change

func (Model) Init() tview.Cmd { return nil }

// View shows the picker in a box titled Attachments.
func (m Model) View() tview.Element {
	p := ui.Picker(m.items, &m.searchState, m.cfg).
		OnChange(func(a picker.Change) tview.Msg { return changeMsg(a) }).
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
	if msg, ok := msg.(changeMsg); ok {
		m.searchState.Apply(picker.Change(msg))
	}
	return m, nil
}
