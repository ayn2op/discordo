// Package form lays out labeled text inputs above a submit button.
package form

import (
	"slices"

	"github.com/ayn2op/tview"
	"github.com/ayn2op/tview/button"
	"github.com/ayn2op/tview/column"
	"github.com/ayn2op/tview/row"
	"github.com/ayn2op/tview/text"
	"github.com/ayn2op/tview/textinput"
	"github.com/gdamore/tcell/v3"
	"github.com/rivo/uniseg"
)

// Field is a labeled text input. A non-empty Mask hides what is typed, such as for passwords.
type Field struct {
	Label, Mask string
	EditState   textinput.EditState
}

// Model is a column of fields above a button. Tab and the arrow keys move the focus, Enter in a field moves to the next one, and Enter on the button or clicking it submits.
type Model struct {
	fields []Field
	button string
	focus  int
}

// SubmitMsg is returned when the button is pressed.
type SubmitMsg struct{}

type editMsg struct {
	index  int
	change textinput.Change
}

type nextMsg struct{}

func New(button string, fields ...Field) Model {
	return Model{fields: fields, button: button}
}

var _ tview.Model[Model] = Model{}

// Value returns the text of field index.
func (m Model) Value(index int) string {
	return m.fields[index].EditState.Value()
}

func (Model) Init() tview.Cmd { return nil }

func (m Model) Update(msg tview.Msg) (Model, tview.Cmd) {
	count := len(m.fields) + 1
	switch msg := msg.(type) {
	case editMsg:
		// Earlier copies of the model share the fields, so change a copy of them.
		m.fields = slices.Clone(m.fields)
		m.fields[msg.index].EditState.Apply(msg.change)
	case nextMsg:
		m.focus = (m.focus + 1) % count
	case tview.KeyMsg:
		switch msg.Key() {
		case tcell.KeyTab, tcell.KeyDown:
			m.focus = (m.focus + 1) % count
		case tcell.KeyBacktab, tcell.KeyUp:
			m.focus = (m.focus + count - 1) % count
		}
	}
	return m, nil
}

func (m Model) View() tview.Element {
	labelWidth := 0
	for _, f := range m.fields {
		labelWidth = max(labelWidth, uniseg.StringWidth(f.Label)+1)
	}
	layout := column.New().Spacing(1)
	for i := range m.fields {
		f := &m.fields[i]
		input := textinput.New(&f.EditState).
			Mask(f.Mask).
			Focused(m.focus == i).
			OnChange(func(a textinput.Change) tview.Msg { return editMsg{index: i, change: a} }).
			OnSubmit(nextMsg{})
		layout = layout.Push(row.New(column.New(text.New(f.Label)).Width(tview.Fixed(labelWidth)), input).Height(tview.Fixed(1)))
	}
	submit := button.New().
		Label(m.button).
		Width(tview.Fixed(uniseg.StringWidth(m.button) + 4)).
		Height(tview.Fixed(1)).
		Focused(m.focus == len(m.fields)).
		OnClick(SubmitMsg{})
	return layout.Push(submit)
}
