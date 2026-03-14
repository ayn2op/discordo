package root

import (
	"log/slog"

	"github.com/ayn2op/discordo/internal/clipboard"
	"github.com/ayn2op/discordo/internal/keyring"
	"github.com/ayn2op/tview"
	"github.com/gdamore/tcell/v3"
)

type tokenEvent struct {
	tcell.EventTime
	token string
}

func tokenCommand(token string) tview.Command {
	return func() tcell.Event {
		event := &tokenEvent{token: token}
		event.SetEventNow()
		return event
	}
}

type loginEvent struct{ tcell.EventTime }

func getToken() tview.Command {
	return func() tcell.Event {
		token, err := keyring.GetToken()
		if err != nil {
			slog.Info("failed to retrieve token from keyring", "err", err)
			event := &loginEvent{}
			event.SetEventNow()
			return event
		}
		event := &tokenEvent{token: token}
		event.SetEventNow()
		return event
	}
}

func setToken(token string) tview.Command {
	return func() tcell.Event {
		if err := keyring.SetToken(token); err != nil {
			slog.Error("failed to set token to keyring", "err", err)
			return tcell.NewEventError(err)
		}
		return nil
	}
}

func deleteToken() tview.Command {
	return func() tcell.Event {
		if err := keyring.DeleteToken(); err != nil {
			slog.Error("failed to delete token from keyring", "err", err)
			return tcell.NewEventError(err)
		}
		return nil
	}
}

func initClipboard() tview.Command {
	return func() tcell.Event {
		if err := clipboard.Init(); err != nil {
			slog.Error("failed to init clipboard", "err", err)
			return tcell.NewEventError(err)
		}
		return nil
	}
}
