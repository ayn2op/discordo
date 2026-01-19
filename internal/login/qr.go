package login

import (
	"charm.land/bubbletea/v2"
	"github.com/ayn2op/discordo/pkg/tabs"
	"github.com/coder/websocket"
)

type QRModel struct {
	conn *websocket.Conn
}

func newQRModel() QRModel {
	return QRModel{}
}

var _ tabs.Tab = QRModel{}

func (m QRModel) Name() string {
	return "QR"
}

var _ tea.Model = QRModel{}

func (m QRModel) Init() tea.Cmd {
	return nil
}

func (m QRModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return m, nil
}

func (m QRModel) View() tea.View {
	return tea.NewView("")
}
