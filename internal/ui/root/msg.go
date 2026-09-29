package root

import (
	"log/slog"

	"github.com/ayn2op/discordo/internal/consts"
	"github.com/ayn2op/tview"
	"github.com/zalando/go-keyring"
	"golang.design/x/clipboard"
)

const (
	keyringService = consts.Name
	keyringUser    = "token"
)

type tokenMsg string

func useToken(token string) tview.Cmd {
	return func() tview.Msg {
		return tokenMsg(token)
	}
}

type loginMsg struct{}

func getToken() tview.Cmd {
	return func() tview.Msg {
		token, err := keyring.Get(keyringService, keyringUser)
		if err != nil {
			slog.Info("failed to retrieve token from keyring", "err", err)
			return loginMsg{}
		}
		return tokenMsg(token)
	}
}

func setToken(token string) tview.Cmd {
	return func() tview.Msg {
		if err := keyring.Set(keyringService, keyringUser, token); err != nil {
			slog.Error("failed to set token to keyring", "err", err)
			return nil
		}
		return nil
	}
}

func deleteToken() tview.Cmd {
	return func() tview.Msg {
		if err := keyring.Delete(keyringService, keyringUser); err != nil {
			slog.Error("failed to delete token from keyring", "err", err)
			return nil
		}
		return nil
	}
}

func initClipboard() tview.Cmd {
	return func() tview.Msg {
		if err := clipboard.Init(); err != nil {
			slog.Error("failed to init clipboard", "err", err)
			return nil
		}
		return nil
	}
}
