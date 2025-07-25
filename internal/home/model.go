package home

import (
	"context"
	"log"

	"github.com/ayn2op/discordo/internal/guildstree"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/diamondburned/ningen/v3"
)

type Model struct {
	state  *ningen.State
	events chan tea.Msg

	guildstree guildstree.Model
}

func NewModel(token string) Model {
	state := ningen.New(token)
	events := make(chan tea.Msg)

	state.AddHandler(func(event any) {
		events <- event
	})

	return Model{
		state:  state,
		events: events,

		guildstree: guildstree.NewModel(state),
	}
}

func (m Model) Init() tea.Cmd {
	go m.state.Open(context.TODO())
	return m.listen
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case *gateway.ReadyEvent:
		for _, folder := range msg.UserSettings.GuildFolders {
			if folder.ID == 0 && len(folder.GuildIDs) == 1 {
				guild, err := m.state.Guild(folder.GuildIDs[0])
				if err != nil {
					log.Fatal(err)
				}

				m.guildstree.AddGuild(*guild)
			} else {
				m.guildstree.AddFolder(folder)
			}
		}
	}

	return m, m.listen
}

func (m Model) View() string {
	return m.guildstree.View()
}

func (m Model) listen() tea.Msg {
	return <-m.events
}
