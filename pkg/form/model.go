package form

import (
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

const maxInputWidth = 60

type Model struct {
	KeyMap    KeyMap
	Submitted bool

	inputs []textinput.Model
	active int
	width  int
	height int
}

func NewModel(inputs []textinput.Model) *Model {
	return &Model{
		KeyMap: DefaultKeyMap(),
		inputs: inputs,
	}
}

func (m *Model) Get(index int) textinput.Model {
	return m.inputs[index]
}

func (m *Model) Init() tea.Cmd {
	for i := range m.inputs {
		m.inputs[i].SetValue("")
	}

	return tea.RequestWindowSize
}

func (m *Model) Update(msg tea.Msg) (*Model, tea.Cmd) {
	if len(m.inputs) == 0 {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		width := min(msg.Width, maxInputWidth)
		for i := range m.inputs {
			m.inputs[i].SetWidth(width)
		}
	case tea.KeyMsg:
		k := msg.Key()
		switch {
		case key.Matches(k, m.KeyMap.Previous):
			m.active = max(m.active-1, 0)
		case key.Matches(k, m.KeyMap.Next):
			m.active = min(m.active+1, len(m.inputs)-1)
		case key.Matches(k, m.KeyMap.Submit):
			if m.active == len(m.inputs)-1 {
				if m.inputs[m.active].Value() != "" {
					m.Submitted = true
				}
			}
		}
	}

	var cmds []tea.Cmd
	for i := range m.inputs {
		if i == m.active {
			updated, cmd := m.inputs[i].Update(msg)
			m.inputs[i] = updated
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
			cmds = append(cmds, m.inputs[i].Focus())
		} else {
			m.inputs[i].Blur()
		}
	}

	return m, tea.Batch(cmds...)
}

func (m *Model) View() tea.View {
	views := make([]string, len(m.inputs))
	for i, input := range m.inputs {
		views[i] = input.View()
	}

	form := strings.Join(views, "\n")
	centered := lipgloss.Place(
		m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		form,
	)
	return tea.NewView(centered)
}
