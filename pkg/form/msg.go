package form

import tea "charm.land/bubbletea/v2"

type SubmitMsg struct{}

func submit() tea.Cmd {
	return func() tea.Msg {
		return SubmitMsg{}
	}
}
