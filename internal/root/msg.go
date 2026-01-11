package root

import (
	"os"

	tea "charm.land/bubbletea/v2"
	"github.com/ayn2op/discordo/internal/keyring"
	"github.com/ayn2op/discordo/internal/login"
)

func getToken() tea.Msg {
	token := os.Getenv("DISCORDO_TOKEN")
	if token != "" {
		return login.TokenMsg{Value: token, Err: nil}
	}
	token, err := keyring.GetToken()
	return login.TokenMsg{Value: token, Err: err}
}
