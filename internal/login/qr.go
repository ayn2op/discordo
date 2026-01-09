package login

import tea "charm.land/bubbletea/v2"

type QRModel struct{}

func newQRModel() QRModel {
	return QRModel{}
}

var _ tea.Model = QRModel{}

func (m QRModel) Init() tea.Cmd {
	return nil
}

func (m QRModel) Update(tea.Msg) (tea.Model, tea.Cmd) {
	return m, nil
}

func (m QRModel) View() tea.View {
	return tea.NewView("qr")
}
