// Package reminderprompt asks when to be reminded of a message, as typed.
package reminderprompt

import (
	"strings"

	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/discordo/internal/ui"
	"github.com/ayn2op/tview"
	"github.com/ayn2op/tview/keybind"
	"github.com/ayn2op/tview/textinput"
)

const (
	title = "Remind me at"
	hint  = "45m, 18:30, 2026-11-02 09:00"
)

// Model is a one-line input. Enter submits what is typed, the picker's cancel key cancels.
type Model struct {
	cfg  *config.Config
	text string
	// cursor is the byte offset of the cursor in text.
	cursor int
	// err is why the last submit was rejected, shown in place of the hint, or nil if nothing was rejected.
	err error
	// onSubmit turns what is typed into the command the prompt was opened for, or into the error to show instead of closing.
	onSubmit func(text string) (tview.Cmd, error)
}

func NewModel(cfg *config.Config) Model {
	return Model{cfg: cfg}
}

var _ tview.Model[Model] = Model{}

func (Model) Init() tview.Cmd { return nil }

// Open empties the input and sets what submitting does.
func (m *Model) Open(onSubmit func(text string) (tview.Cmd, error)) {
	m.text, m.cursor, m.err, m.onSubmit = "", 0, nil, onSubmit
}

func (m Model) View() tview.Widget {
	input := textinput.New(m.text, m.cursor).
		Focused(true).
		OnInput(func(text string, cursor int) tview.Msg { return inputMsg{text, cursor} }).
		OnSubmit(submitMsg{})
	return ui.Box(input, m.cfg, true).
		Title(ui.Label(title, m.cfg.Theme.Title, true)).
		Footer(ui.Label(m.footer(), m.cfg.Theme.Footer, true))
}

// footer is what the input takes, after why the last submit was rejected if one was, as that is when the examples are wanted most.
func (m Model) footer() string {
	if m.err != nil {
		return m.err.Error() + ": " + hint
	}
	return hint
}

func (m Model) Update(msg tview.Msg) (Model, tview.Cmd) {
	switch msg := msg.(type) {
	case inputMsg:
		// Editing the text drops what it was rejected for, bringing the hint back.
		m.text, m.cursor, m.err = msg.text, msg.cursor, nil
	case submitMsg:
		return m.submit()
	case tview.KeyMsg:
		if keybind.Matches(msg, m.cfg.Keybinds.Picker.Cancel.Keybind) {
			return m, func() tview.Msg { return CancelMsg{} }
		}
	}
	return m, nil
}

// submit closes the prompt with what was typed, or keeps it open showing why that was rejected.
// An empty input does nothing, as there is nothing to submit.
func (m Model) submit() (Model, tview.Cmd) {
	if strings.TrimSpace(m.text) == "" {
		return m, nil
	}

	action, err := m.onSubmit(m.text)
	if err != nil {
		m.err = err
		return m, nil
	}

	return m, func() tview.Msg { return SubmittedMsg{Action: action} }
}
