package cmd

import (
	"log/slog"

	"github.com/ayn2op/discordo/internal/constants"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/zalando/go-keyring"
)

type Layout struct {
	app  *tview.Application
	flex *tview.Flex

	guildsTree   *GuildsTree
	messagesText *MessagesText
	messageInput *MessageInput
}

func newLayout() *Layout {
	app := tview.NewApplication()
	l := &Layout{
		app:  app,
		flex: tview.NewFlex(),

		guildsTree:   newGuildsTree(app),
		messagesText: newMessagesText(app),
		messageInput: newMessageInput(app),
	}

	l.init()

	l.app.EnableMouse(cfg.Mouse)
	l.app.SetInputCapture(l.onAppInputCapture)

	l.flex.SetInputCapture(l.onFlexInputCapture)
	return l
}

func (l *Layout) show(token string) error {
	if token == "" {
		loginForm := newLoginForm(func(token string, err error) {
			if err != nil {
				slog.Error("failed to login", "err", err)
				return
			}

			if err := l.show(token); err != nil {
				slog.Error("failed to show app", "err", err)
			}
		})
		l.app.SetRoot(loginForm, true)
	} else {
		if err := openState(token, l.app); err != nil {
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
	case cfg.Keys.Quit:
		l.app.Stop()
	case "Ctrl+C":
		// https://github.com/rivo/tview/blob/a64fc48d7654432f71922c8b908280cdb525805c/application.go#L153
		return tcell.NewEventKey(tcell.KeyCtrlC, 0, tcell.ModNone)
	}

	return event
}

func (l *Layout) onFlexInputCapture(event *tcell.EventKey) *tcell.EventKey {
	switch event.Name() {
	case cfg.Keys.FocusGuildsTree:
		l.app.SetFocus(l.guildsTree)
		return nil
	case cfg.Keys.FocusMessagesText:
		l.app.SetFocus(l.messagesText)
		return nil
	case cfg.Keys.FocusMessageInput:
		l.app.SetFocus(l.messageInput)
		return nil
	case cfg.Keys.Logout:
		l.app.Stop()

		if err := keyring.Delete(constants.Name, "token"); err != nil {
			slog.Error("failed to delete token from keyring", "err", err)
			return nil
		}

		return nil
	case cfg.Keys.ToggleGuildsTree:
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
