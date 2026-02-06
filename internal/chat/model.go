package chat

import (
	"github.com/ayn2op/discordo/pkg/tea"
)

type Model struct{}

func NewModel() Model {
	return Model{}
}

var _ tea.Model = Model{}

func (m Model) Init() tea.Cmd {
	panic("unimplemented")
}

func (m Model) Update(tea.Msg) (tea.Model, tea.Cmd) {
	panic("unimplemented")
}

func (m Model) View(frame *tea.Frame, area tea.Rect) {
	frame.PutStr(area.Min.X, area.Min.Y, "Chat")
}
