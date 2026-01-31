package root

import (
	"os"

	tea "charm.land/bubbletea/v2"
	"github.com/ayn2op/discordo/internal/keyring"
	"github.com/ayn2op/discordo/internal/login/token"
)

func getToken() tea.Msg {
	t := os.Getenv("DISCORDO_TOKEN")
	if t != "" {
		return token.TokenMsg{Value: t, Err: nil}
	}
	t, err := keyring.GetToken()
	return token.TokenMsg{Value: t, Err: err}
}
