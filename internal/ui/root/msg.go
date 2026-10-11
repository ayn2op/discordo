package root

import (
	"log/slog"

	"github.com/ayn2op/discordo/internal/consts"
	"github.com/ayn2op/discordo/internal/ui/chat"
	"github.com/ayn2op/discordo/internal/ui/login"
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
		}
		return nil
	}
}

func deleteToken() tview.Cmd {
	return func() tview.Msg {
		if err := keyring.Delete(keyringService, keyringUser); err != nil {
			slog.Error("failed to delete token from keyring", "err", err)
		}
		return nil
	}
}

func initClipboard() tview.Cmd {
	return func() tview.Msg {
		if err := clipboard.Init(); err != nil {
			slog.Error("failed to init clipboard", "err", err)
		}
		return nil
	}
}

func (m *Model) showLogin() tview.Cmd {
	m.screen, m.login, m.chat = screenLogin, login.NewModel(m.cfg), chat.Model{}
	return m.show(m.login.Init())
}

func (m *Model) showChat(token string) tview.Cmd {
	m.screen, m.chat, m.login = screenChat, chat.NewModel(m.cfg, token), login.Model{}
	init := m.chat.Init()
	if m.resize != nil {
		init = tview.Batch(init, m.updateScreen(m.resize))
	}
	return m.show(init)
}

// show closes any open dialog after the shown model changed and returns what it needs done, with init its first command.
func (m *Model) show(init tview.Cmd) tview.Cmd {
	m.dialog = nil
	return tview.Batch(tview.SetTitle(consts.Name), init)
}

func (m *Model) finishDialog(index int) tview.Cmd {
	if m.dialog == nil || index < 0 || index >= len(m.dialog.Buttons) {
		m.dialog = nil
		return nil
	}
	button := m.dialog.Buttons[index]
	if !button.KeepOpen {
		m.dialog = nil
	}
	return button.Cmd
}
