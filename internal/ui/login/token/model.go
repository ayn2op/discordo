package token

import (
	"github.com/ayn2op/tview"
	"github.com/ayn2op/tview/button"
	"github.com/ayn2op/tview/column"
	"github.com/ayn2op/tview/layout"
	"github.com/ayn2op/tview/row"
	"github.com/ayn2op/tview/text"
	"github.com/ayn2op/tview/textinput"
	"github.com/gdamore/tcell/v3"
)

// Model is a token input above a button. Tab and the arrow keys move the focus, Enter in the input moves to the button, and Enter on the button or clicking it logs in.
type Model struct {
	token         textinput.EditState
	buttonFocused bool
}

var _ tview.Model[Model] = Model{}

func (Model) Label() string {
	return "Token"
}

func (Model) Init() tview.Cmd { return nil }

func (m Model) Update(msg tview.Msg) (Model, tview.Cmd) {
	switch msg := msg.(type) {
	case tokenMsg:
		m.token.Apply(textinput.Change(msg))
	case focusButtonMsg:
		m.buttonFocused = true
	case tview.KeyMsg:
		switch msg.Key() {
		case tcell.KeyTab, tcell.KeyDown, tcell.KeyBacktab, tcell.KeyUp:
			m.buttonFocused = !m.buttonFocused
		}
	case submitMsg:
		token := m.token.Value()
		if token == "" {
			return m, nil
		}
		return m, submitToken(token)
	}
	return m, nil
}

func (m Model) View() tview.Element {
	input := textinput.New(&m.token).
		Mask("*").
		Focused(!m.buttonFocused).
		OnChange(func(c textinput.Change) tview.Msg { return tokenMsg(c) }).
		OnSubmit(focusButtonMsg{})
	submit := button.New().
		Label("Login").
		Width(layout.Fixed(len("Login") + 4)).
		Height(layout.Fixed(1)).
		Focused(m.buttonFocused).
		OnClick(submitMsg{})
	return column.New(
		row.New(column.New(text.New("Token")).Width(layout.Fixed(len("Token")+1)), input).Height(layout.Fixed(1)),
		submit,
	).Spacing(1)
}
