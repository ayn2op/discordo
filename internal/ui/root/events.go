package root

import (
	"log/slog"
	"os"

	"github.com/ayn2op/discordo/internal/clipboard"
	"github.com/ayn2op/discordo/internal/keyring"
	"github.com/gdamore/tcell/v3"
)

type tokenEvent struct {
	tcell.EventTime
	token string
}

func newTokenEvent(token string) *tokenEvent {
	event := &tokenEvent{token: token}
	event.SetEventNow()
	return event
}

func getToken() tcell.Event {
	token := os.Getenv(tokenEnvVarKey)
	if token == "" {
		tok, err := keyring.GetToken()
		if err != nil {
			slog.Info("failed to retrieve token from keyring", "err", err)
		}
		token = tok
	}
	return newTokenEvent(token)
}

func deleteToken() tcell.Event {
	if err := keyring.DeleteToken(); err != nil {
		slog.Error("failed to delete token from keyring", "err", err)
		return tcell.NewEventError(err)
	}
	return nil
}

func initClipboard() tcell.Event {
	if err := clipboard.Init(); err != nil {
		slog.Error("failed to init clipboard", "err", err)
		return tcell.NewEventError(err)
	}
	return nil
}
