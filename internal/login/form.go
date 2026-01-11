package login

import (
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type FormModel struct {
	inputs []textinput.Model
	active int
	width  int
	height int
}

const maxInputWidth = 60

func newFormModel(inputs []textinput.Model) *FormModel {
	return &FormModel{inputs: inputs}
}

func (m *FormModel) Init() tea.Cmd {
	if m.active >= len(m.inputs) {
		m.active = len(m.inputs) - 1
	}

	return tea.RequestWindowSize
}

func (m *FormModel) Update(msg tea.Msg) (*FormModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		width := min(msg.Width, maxInputWidth)
		if width > 0 {
			for i := range m.inputs {
				m.inputs[i].SetWidth(width)
			}
		}
	case tea.KeyMsg:
		if len(m.inputs) > 0 {
			switch msg.String() {
			case "tab":
				if m.active < len(m.inputs)-1 {
					m.active++
				}
			case "shift+tab":
				if m.active > 0 {
					m.active--
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

func (m *FormModel) View() tea.View {
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
