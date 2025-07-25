package guildstree

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/diamondburned/ningen/v3"
)

type Model struct {
	state *ningen.State
}

func NewModel(state *ningen.State) Model {
	return Model{
		state: state,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	return m, nil
}

func (m Model) View() string {
	return "tree"
}

func (m *Model) AddFolder(folder gateway.GuildFolder) {}

func (m *Model) AddGuild(guild discord.Guild) {}
