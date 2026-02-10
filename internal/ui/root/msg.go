package root

import (
	"log/slog"

	tea "charm.land/bubbletea/v2"
	"github.com/ayn2op/discordo/internal/keyring"
)

type (
	tokenMsg string
	loginMsg struct{}
)

func tokenCmd(token string) tea.Cmd {
	return func() tea.Msg {
		return tokenMsg(token)
	}
}

func getToken() tea.Cmd {
	return func() tea.Msg {
		token, err := keyring.GetToken()
		if err != nil {
			slog.Info("failed to get token from keyring", "err", err)
			return loginMsg{}
		}
		return tokenCmd(token)()
	}
}

func setToken(token string) tea.Cmd {
	return func() tea.Msg {
		return keyring.SetToken(token)
	}
}

func deleteToken() tea.Cmd {
	return func() tea.Msg {
		return keyring.DeleteToken()
	}
}
