package guildstree

import (
	"github.com/ayn2op/discordo/tree"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/diamondburned/ningen/v3"
)

type Model struct {
	state *ningen.State
	tree  tree.Model
}

func NewModel(state *ningen.State) Model {
	return Model{
		state: state,
		tree:  tree.NewModel(),
	}
}

func (m Model) Init() tea.Cmd {
	return m.tree.Init()
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd
	m.tree, cmd = m.tree.Update(msg)
	return m, cmd
}

func (m Model) View() string {
	return m.tree.View()
}

func (m *Model) AddFolder(folder gateway.GuildFolder) {
	folderNode := &folderNode{GuildFolder: folder}
	m.tree.AppendNode(folderNode)

	for _, guildID := range folder.GuildIDs {
		guild, err := m.state.Cabinet.Guild(guildID)
		if err != nil {
			// TODO: handle error
			panic(err)
		}

		m.AddGuild(folderNode, *guild)
	}
}

func (m *Model) AddGuild(node *folderNode, guild discord.Guild) {
	guildNode := &guildNode{Guild: guild}
	if node == nil {
		m.tree.AppendNode(guildNode)
	} else {
		node.AppendChild(guildNode)
	}
}
