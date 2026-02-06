package tabs

import (
	"strings"

	"github.com/ayn2op/discordo/pkg/tea"
)

type Tab interface {
	tea.Model
	Label() string
}

type Model struct {
	Keybinds Keybinds
	tabs     []Tab
	active   int
}

func NewModel(tabs []Tab) Model {
	return Model{
		Keybinds: DefaultKeybinds(),
		tabs:     tabs,
	}
}

func (m Model) Init() tea.Cmd {
	if len(m.tabs) == 0 {
		return nil
	}
	return m.tabs[m.active].Init()
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if len(m.tabs) == 0 {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Name() {
		case m.Keybinds.Next:
			m.active = min(m.active+1, len(m.tabs)-1)
			return m, m.tabs[m.active].Init()
		case m.Keybinds.Previous:
			m.active = max(m.active-1, 0)
			return m, m.tabs[m.active].Init()
		}
	}

	var (
		tabModel tea.Model
		cmd      tea.Cmd
	)
	tabModel, cmd = m.tabs[m.active].Update(msg)
	if updated, ok := tabModel.(Tab); ok {
		m.tabs[m.active] = updated
	}
	return m, cmd
}

func (m Model) View(frame *tea.Frame, area tea.Rect) {
	if len(m.tabs) == 0 {
		return
	}

	var labels strings.Builder
	for i, tab := range m.tabs {
		if i > 0 {
			labels.WriteString(" | ")
		}
		if i == m.active {
			labels.WriteByte('[')
		}
		labels.WriteString(tab.Label())
		if i == m.active {
			labels.WriteByte(']')
		}
	}
	frame.PutStr(area.Min.X, area.Min.Y, labels.String())

	content := area
	content.Min.Y++
	if content.Dy() <= 0 {
		return
	}
	m.tabs[m.active].View(frame, content)
}
