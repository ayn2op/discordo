package tree

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type Node interface {
	Name() string
	Children() []Node

	Expanded() bool
	SetExpanded(bool)
}

type Model struct {
	nodes  []Node
	cursor int
}

func NewModel() Model {
	return Model{}
}

func (m *Model) AppendNode(node Node) {
	m.nodes = append(m.nodes, node)
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	flat := flatten(m.nodes, 0)
	if len(flat) == 0 {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyUp:
			m.cursor = max(m.cursor-1, 0)
		case tea.KeyDown:
			m.cursor = min(m.cursor+1, len(flat)-1)
		case tea.KeyEnter:
			node := flat[m.cursor]
			node.SetExpanded(!node.Expanded())
		}
	}

	return m, nil
}

func (m Model) View() string {
	var b strings.Builder

	nodes := flatten(m.nodes, 0)
	for i, node := range nodes {
		cursor := "  "
		if i == m.cursor {
			cursor = "> "
		}

		indent := strings.Repeat("  ", node.depth)
		fmt.Fprintf(&b, "%s%s %s\n", cursor, indent, node.Name())
	}

	return b.String()
}

type flatNode struct {
	Node
	depth int
}

func flatten(nodes []Node, depth int) []flatNode {
	var flat []flatNode
	for _, node := range nodes {
		flat = append(flat, flatNode{node, depth})
		if node.Expanded() {
			flat = append(flat, flatten(node.Children(), depth+1)...)
		}
	}

	return flat
}
