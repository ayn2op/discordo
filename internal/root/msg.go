package root

import (
	tea "charm.land/bubbletea/v2"
	"github.com/ayn2op/discordo/internal/keyring"
)

type tokenMsg struct {
	value string
	err   error
}

func getToken() tea.Msg {
	token, err := keyring.GetToken()
	return tokenMsg{value: token, err: err}
}
