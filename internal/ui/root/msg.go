package root

import (
	"log/slog"

	"github.com/ayn2op/discordo/internal/consts"
	"github.com/ayn2op/discordo/internal/ui/chat"
	"github.com/ayn2op/discordo/internal/ui/login"
	"github.com/ayn2op/tview"
	"github.com/ayn2op/tview/cmd"
	"golang.design/x/clipboard"
)

func (m *Model) quit() tview.Cmd {
	if m.screen != screenChat {
		return cmd.Quit{}
	}
	return m.updateScreen(chat.QuitMsg{})
}

func initClipboard() tview.Cmd {
	return cmd.Task(func() tview.Msg {
		if err := clipboard.Init(); err != nil {
			slog.Error("failed to init clipboard", "err", err)
		}
		return nil
	})
}

func (m *Model) showLogin() tview.Cmd {
	m.screen, m.login, m.chat = screenLogin, login.NewModel(m.cfg), chat.Model{}
	return m.show(m.login.Init())
}

func (m *Model) showChat(token string) tview.Cmd {
	m.screen, m.chat, m.login = screenChat, chat.NewModel(m.cfg, token), login.Model{}
	init := m.chat.Init()
	if m.resize != nil {
		init = cmd.Batch{init, m.updateScreen(m.resize)}
	}
	return m.show(init)
}

// show closes any open dialog after the shown model changed and returns what it needs done, with init its first command.
func (m *Model) show(init tview.Cmd) tview.Cmd {
	m.dialog = nil
	return cmd.Batch{cmd.SetTitle(consts.Name), init}
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
