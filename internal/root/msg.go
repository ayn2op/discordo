package root

import (
	"os"

	"github.com/ayn2op/discordo/internal/keyring"
	"github.com/ayn2op/discordo/pkg/tea"
)

type (
	tokenMsg string
	loginMsg struct{}
)

func getToken() tea.Msg {
	t := os.Getenv("DISCORDO_TOKEN")
	if t != "" {
		return tokenMsg(t)
	}
	t, err := keyring.GetToken()
	if err != nil {
		return loginMsg{}
	}
	return tokenMsg(t)
}
