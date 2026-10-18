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
	token string
	// cursor is the byte offset of the cursor in token.
	cursor int

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
		m.token, m.cursor = msg.text, msg.cursor
	case focusButtonMsg:
		m.buttonFocused = true
	case tview.KeyMsg:
		switch msg.Key() {
		case tcell.KeyTab, tcell.KeyDown, tcell.KeyBacktab, tcell.KeyUp:
			m.buttonFocused = !m.buttonFocused
		}
	case submitMsg:
		if m.token == "" {
			return m, nil
		}
		return m, submitToken(m.token)
	}
	return m, nil
}

func (m Model) View() tview.Widget {
	input := textinput.New(m.token, m.cursor).
		Mask("*").
		Focused(!m.buttonFocused).
		OnInput(func(text string, cursor int) tview.Msg { return tokenMsg{text, cursor} }).
		OnSubmit(focusButtonMsg{})
	submit := button.New(text.New("Login")).
		Width(layout.Fixed(len("Login") + 4)).
		Focused(m.buttonFocused).
		OnClick(submitMsg{})
	return column.New(
		row.New(column.New(text.New("Token")).Width(layout.Fixed(len("Token")+1)), input).Height(layout.Shrink),
		submit,
	).Spacing(1).AlignX(tview.HorizontalAlignmentCenter)
}
