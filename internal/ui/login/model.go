package login

import (
	"log/slog"

	"github.com/ayn2op/tview/layers"
	"github.com/ayn2op/tview/modal"
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

	cfg             *config.Config
	errorDialogText string
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

func (m *Model) Update(msg tview.Msg) tview.Cmd {
	switch msg := msg.(type) {
	case errMsg:
		if m.HasLayer(errorLayerName) {
			return nil
		}
		return m.showErrorDialog(msg.err)
	case modal.DoneMsg:
		if !m.HasLayer(errorLayerName) {
			break
		}
		if msg.ButtonIndex == 0 {
			return setClipboard(m.errorDialogText)
		}
		m.RemoveLayer(errorLayerName)
		m.errorDialogText = ""
		return nil
	}
	return m.Layers.Update(msg)
}

func (m *Model) ModalActive() bool {
	return m.HasLayer(errorLayerName)
}

func (m *Model) showErrorDialog(err error) tview.Cmd {
	slog.Error("failed to login", "err", err)

	message := err.Error()
	m.errorDialogText = message
	dialog := modal.NewModel().SetText(message).AddButtons([]string{"Copy", "Close"})
	{
		bg := m.cfg.Theme.Dialog.Style.GetBackground()
		buttonStyle := m.cfg.Theme.Dialog.Style.Style
		if bg != tcell.ColorDefault {
			dialog.SetBackgroundColor(bg)
			buttonStyle = buttonStyle.Background(bg)
		}
		fg := m.cfg.Theme.Dialog.Style.GetForeground()
		if fg != tcell.ColorDefault {
			dialog.SetTextColor(fg)
			buttonStyle = buttonStyle.Foreground(fg)
		}
		dialog.SetButtonStyle(buttonStyle).SetButtonActivatedStyle(buttonStyle.Reverse(true))
	}
	m.
		AddLayer(
			dialog,
			layers.WithName(errorLayerName),
			layers.WithResize(true),
			layers.WithVisible(true),
			layers.WithOverlay(),
		).
		SendToFront(errorLayerName)
	return tview.SetFocus(dialog)
}
