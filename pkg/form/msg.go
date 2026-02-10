package form

import tea "charm.land/bubbletea/v2"

type SubmitMsg struct {
	Values []string
}

func (m Model) submit() tea.Cmd {
	values := make([]string, len(m.inputs))
	for i, input := range m.inputs {
		values[i] = input.Value()
	}
	return func() tea.Msg {
		return SubmitMsg{values}
	}
}

type resetMsg struct{}

func reset() tea.Msg {
	return resetMsg{}
}
