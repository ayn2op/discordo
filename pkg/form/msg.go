package form

import tea "charm.land/bubbletea/v2"

type SubmitMsg struct {
	Values []string
}

func submit(values []string) tea.Cmd {
	return func() tea.Msg {
		return SubmitMsg{Values: values}
	}
}
