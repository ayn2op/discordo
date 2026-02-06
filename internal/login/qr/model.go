package qr

import (
	"github.com/ayn2op/discordo/pkg/tea"
)

type Model struct{}

func NewModel() Model {
	return Model{}
}

var _ tea.Model = Model{}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(tea.Msg) (tea.Model, tea.Cmd) {
	return m, nil
}

func (m Model) View(frame *tea.Frame, area tea.Rect) {
	frame.PutStr(area.Min.X, area.Min.Y, "QR login tab")
}

func (m Model) Label() string {
	return "QR"
}
