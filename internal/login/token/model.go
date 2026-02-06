package token

import (
	"github.com/ayn2op/discordo/pkg/tea"
)

type Model struct{}

func NewModel() Model {
	return Model{}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(tea.Msg) (tea.Model, tea.Cmd) {
	return m, nil
}

func (m Model) View(frame *tea.Frame, area tea.Rect) {
	frame.PutStr(area.Min.X, area.Min.Y, "Token login tab")
}

func (m Model) Label() string {
	return "Token"
}

var _ tea.Model = Model{}
