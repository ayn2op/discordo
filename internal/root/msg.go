package root

import (
	"os"

	tea "charm.land/bubbletea/v2"
	"github.com/ayn2op/discordo/internal/keyring"
)

type TokenMsg struct {
	Value string
	err   error
}

func getToken() tea.Msg {
	token := os.Getenv("DISCORDO_TOKEN")
	if token != "" {
		return TokenMsg{token, nil}
	}
	token, err := keyring.GetToken()
	return TokenMsg{Value: token, err: err}
}
