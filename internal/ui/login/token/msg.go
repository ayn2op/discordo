package token

import tea "charm.land/bubbletea/v2"

type TokenMsg string

func tokenCmd(token string) tea.Cmd {
	return func() tea.Msg {
		return TokenMsg(token)
	}
}
