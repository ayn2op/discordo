package cmd

import (
	"log/slog"

	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/discordo/internal/consts"
	"github.com/ayn2op/discordo/internal/login"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/zalando/go-keyring"
)

type Layout struct {
	cfg          *config.Config
	app          *tview.Application
	flex         *tview.Flex
	guildsTree   *GuildsTree
	messagesText *MessagesText
	messageInput *MessageInput
}

func newLayout(cfg *config.Config) *Layout {
	app := tview.NewApplication()
	l := &Layout{
		cfg:  cfg,
		app:  app,
		flex: tview.NewFlex(),

		guildsTree:   newGuildsTree(app, cfg),
		messagesText: newMessagesText(app, cfg),
		messageInput: newMessageInput(app, cfg),
	}

	l.init()

	l.app.EnableMouse(cfg.Mouse)
	l.app.SetInputCapture(l.onAppInputCapture)

	l.flex.SetInputCapture(l.onFlexInputCapture)
	return l
}

func (l *Layout) show(token string) error {
	if token == "" {
		loginForm := login.NewForm(l.app, func(token string) {
			if err := l.show(token); err != nil {
				slog.Error("failed to show app", "err", err)
				return
			}
		})

		l.app.SetRoot(loginForm, true)
	} else {
		if err := openState(token, l.app, l.cfg); err != nil {
			return err
		}

		l.app.SetRoot(l.flex, true)
	}

	return nil
}

func (l *Layout) run(token string) error {
	if err := l.show(token); err != nil {
		return err
	}

	return l.app.Run()
}

func (l *Layout) init() {
	l.flex.Clear()

	right := tview.NewFlex()
	right.SetDirection(tview.FlexRow)
	right.AddItem(l.messagesText, 0, 1, false)
	right.AddItem(l.messageInput, 3, 1, false)
	// The guilds tree is always focused first at start-up.
	l.flex.AddItem(l.guildsTree, 0, 1, true)
	l.flex.AddItem(right, 0, 4, false)
}

func (l *Layout) onAppInputCapture(event *tcell.EventKey) *tcell.EventKey {
	switch event.Name() {
	case l.cfg.Keys.Quit:
		if discordState != nil {
			if err := discordState.Close(); err != nil {
				slog.Error("failed to close the session", "err", err)
			}
		}

		l.app.Stop()
	case "Ctrl+C":
		// https://github.com/rivo/tview/blob/a64fc48d7654432f71922c8b908280cdb525805c/application.go#L153
		return tcell.NewEventKey(tcell.KeyCtrlC, 0, tcell.ModNone)
	}

	return event
}

func (l *Layout) onFlexInputCapture(event *tcell.EventKey) *tcell.EventKey {
	switch event.Name() {
	case l.cfg.Keys.FocusGuildsTree:
		l.app.SetFocus(l.guildsTree)
		return nil
	case l.cfg.Keys.FocusMessagesText:
		l.app.SetFocus(l.messagesText)
		return nil
	case l.cfg.Keys.FocusMessageInput:
		l.app.SetFocus(l.messageInput)
		return nil
	case l.cfg.Keys.Logout:
		l.app.Stop()

		if err := keyring.Delete(consts.Name, "token"); err != nil {
			slog.Error("failed to delete token from keyring", "err", err)
			return nil
		}

		return nil
	case l.cfg.Keys.ToggleGuildsTree:
		// The guilds tree is visible if the numbers of items is two.
		if l.flex.GetItemCount() == 2 {
			l.flex.RemoveItem(l.guildsTree)

			if l.guildsTree.HasFocus() {
				l.app.SetFocus(l.flex)
			}
		} else {
			l.init()
			l.app.SetFocus(l.guildsTree)
		}

		return nil
	}

	return event
}
