package form

import (
	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

const (
	maxInputWidth = 64
)

type Model struct {
	Keybinds Keybinds

	inputs []textinput.Model
	active int

	windowSize tea.WindowSizeMsg
}

func NewModel(inputs []textinput.Model) Model {
	m := Model{
		Keybinds: DefaultKeybinds(),
		inputs:   inputs,
	}
	return m
}

func (m *Model) Previous() {
	if len(m.inputs) == 0 {
		return
	}
	m.inputs[m.active].Blur()
	m.active = max(m.active-1, 0)
	m.inputs[m.active].Focus()
}

func (m *Model) Next() {
	if len(m.inputs) == 0 {
		return
	}
	m.inputs[m.active].Blur()
	m.active = min(m.active+1, len(m.inputs)-1)
	m.inputs[m.active].Focus()
}

func (m Model) Init() tea.Cmd {
	if len(m.inputs) == 0 {
		return nil
	}
	return tea.Batch(textinput.Blink, reset)
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if len(m.inputs) == 0 {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.windowSize = msg
		width := min(msg.Width, maxInputWidth)
		for i := range m.inputs {
			m.inputs[i].SetWidth(width)
		}

	case tea.KeyMsg:
		k := msg.Key()
		switch {
		case key.Matches(k, m.Keybinds.Next):
			m.Next()
			return m, nil
		case key.Matches(k, m.Keybinds.Previous):
			m.Previous()
			return m, nil
		case key.Matches(k, m.Keybinds.Submit):
			if m.active < len(m.inputs)-1 {
				m.Next()
				return m, nil
			}
			return m, m.submit()
		}

	case resetMsg:
		for i := range m.inputs {
			m.inputs[i].SetValue("")
			m.inputs[i].Blur()
		}
		m.active = 0
		return m, m.inputs[m.active].Focus()
	}

	var cmd tea.Cmd
	m.inputs[m.active], cmd = m.inputs[m.active].Update(msg)
	return m, cmd
}

func (m Model) View() tea.View {
	if len(m.inputs) == 0 {
		return tea.NewView("")
	}

	views := make([]string, 0, len(m.inputs))
	for i := range m.inputs {
		views = append(views, m.inputs[i].View())
	}
	return tea.NewView(lipgloss.JoinVertical(lipgloss.Left, views...))
}

var _ help.KeyMap = Model{}

func (m Model) ShortHelp() []key.Binding {
	var keybinds []key.Binding

	count := len(m.inputs)
	if count == 0 {
		return keybinds
	}
	// Only show previous keybind when there is an input before the current one.
	if m.active > 0 {
		keybinds = append(keybinds, m.Keybinds.Previous)
	}
	// if the active input is the last input, show the submit keybind.
	if m.active == count-1 {
		keybinds = append(keybinds, m.Keybinds.Submit)
	} else {
		keybinds = append(keybinds, m.Keybinds.Next)
	}
	return keybinds
}

func (m Model) FullHelp() [][]key.Binding {
	short := m.ShortHelp()
	return [][]key.Binding{short}
}
