package login

import (
	"log/slog"

	"github.com/ayn2op/tview/layers"
	"github.com/ayn2op/tview/tabs"

	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/discordo/internal/ui"
	"github.com/ayn2op/discordo/internal/ui/login/password"
	"github.com/ayn2op/discordo/internal/ui/login/qr"
	"github.com/ayn2op/discordo/internal/ui/login/token"
	"github.com/ayn2op/tview"
)

const (
	tabsLayerName = "tabs"
)

type Model struct {
	*layers.Layers
	tabs *tabs.Model
}

func NewModel(cfg *config.Config) *Model {
	tabs := tabs.NewModel([]tabs.Tab{password.NewModel(), qr.NewModel(), token.NewModel()})

	l := layers.New()
	ui.ConfigureBox(l.Box, &cfg.Theme)
	l.SetBackgroundLayerStyle(cfg.Theme.Dialog.BackgroundStyle.Style)
	l.AddLayer(tabs, layers.WithName(tabsLayerName), layers.WithResize(true), layers.WithVisible(true))
	return &Model{
		Layers: l,
		tabs:   tabs,
	}
}

func (m *Model) Update(msg tview.Msg) tview.Cmd {
	switch msg := msg.(type) {
	case errMsg:
		return m.showErrorDialog(msg)
	case password.ErrMsg:
		return m.showErrorDialog(msg)
	case copyErrorMsg:
		return setClipboard(string(msg))
	}
	return m.Layers.Update(msg)
}

func (m *Model) showErrorDialog(err error) tview.Cmd {
	slog.Error("failed to login", "err", err)
	message := err.Error()
	return ui.ShowModal(message,
		ui.ModalButton{Label: "Copy", Result: copyErrorMsg(message), KeepOpen: true},
		ui.ModalButton{Label: "Close"},
	)
}
