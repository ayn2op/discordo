package login

import (
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
)

type FormModel struct {
	inputs []textinput.Model
	active int
}

func newFormModel(inputs []textinput.Model) *FormModel {
	return &FormModel{inputs: inputs}
}

func (m *FormModel) Init() tea.Cmd {
	if m.active >= len(m.inputs) {
		m.active = len(m.inputs) - 1
	}

	return tea.Batch(m.inputs[m.active].Focus(), tea.RequestWindowSize)
}

func (m *FormModel) Update(msg tea.Msg) (*FormModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		for i := range m.inputs {
			m.inputs[i].SetWidth(msg.Width)
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
			var cmd tea.Cmd
			m.inputs[i], cmd = m.inputs[i].Update(msg)
			cmds = append(cmds, cmd)

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

	return tea.NewView(strings.Join(views, "\n"))
}
