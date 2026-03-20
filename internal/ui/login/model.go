package login

import (
	"log/slog"

	"github.com/ayn2op/tview/layers"
	"github.com/ayn2op/tview/tabs"
	"github.com/gdamore/tcell/v3"

	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/discordo/internal/ui"
	"github.com/ayn2op/discordo/internal/ui/login/qr"
	"github.com/ayn2op/discordo/internal/ui/login/token"
	"github.com/ayn2op/tview"
)

const (
	tabsLayerName  = "tabs"
	errorLayerName = "error"
)

type Model struct {
	*layers.Layers
	tabs *tabs.Model

	cfg            *config.Config
	errorModalText string
}

func NewModel(cfg *config.Config) *Model {
	tabs := tabs.NewModel([]tabs.Tab{token.NewModel(), qr.NewModel()})

	l := layers.New()
	ui.ConfigureBox(l.Box, &cfg.Theme)
	l.SetBackgroundLayerStyle(cfg.Theme.Dialog.BackgroundStyle.Style)
	l.AddLayer(tabs, layers.WithName(tabsLayerName), layers.WithResize(true), layers.WithVisible(true))
	return &Model{
		Layers: l,
		tabs:   tabs,
		cfg:    cfg,
	}
}

func (m *Model) HandleEvent(event tview.Event) tview.Command {
	switch event := event.(type) {
	case *tcell.EventError:
		if m.HasLayer(errorLayerName) {
			return nil
		}
		return m.showErrorDialog(event)
	case *tview.ModalDoneEvent:
		if !m.HasLayer(errorLayerName) {
			return nil
		}
		if event.ButtonIndex == 0 {
			return setClipboard(m.errorModalText)
		}
		m.RemoveLayer(errorLayerName)
		m.errorModalText = ""
		return nil
	}
	return m.Layers.HandleEvent(event)
}

func (m *Model) showErrorDialog(err error) tview.Command {
	slog.Error("failed to login", "err", err)

	message := err.Error()
	m.errorModalText = message
	modal := tview.NewModal().
		SetText(message).
		AddButtons([]string{"Copy", "Close"})
	{
		bg := m.cfg.Theme.Dialog.Style.GetBackground()
		buttonStyle := m.cfg.Theme.Dialog.Style.Style
		if bg != tcell.ColorDefault {
			modal.SetBackgroundColor(bg)
			buttonStyle = buttonStyle.Background(bg)
		}
		fg := m.cfg.Theme.Dialog.Style.GetForeground()
		if fg != tcell.ColorDefault {
			modal.SetTextColor(fg)
			buttonStyle = buttonStyle.Foreground(fg)
		}
		// Keep button styles aligned with dialog content and still show focus.
		modal.SetButtonStyle(buttonStyle)
		modal.SetButtonActivatedStyle(buttonStyle.Reverse(true))
	}
	m.
		AddLayer(
			ui.Centered(modal, 0, 0),
			layers.WithName(errorLayerName),
			layers.WithResize(true),
			layers.WithVisible(true),
			layers.WithOverlay(),
		).
		SendToFront(errorLayerName)
	return tview.SetFocus(modal)
}
