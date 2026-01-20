package chat

import (
	"fmt"
	"sort"

	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/discordo/internal/ui"
	"github.com/ayn2op/tview"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/gdamore/tcell/v3"
	"github.com/sahilm/fuzzy"
)

type picker struct {
	*tview.Flex
	view *View
	cfg  *config.Config

	inputField *tview.InputField
	list       *tview.List

	candidates []channelCandidate
}

func newPicker(view *View, cfg *config.Config) *picker {
	p := &picker{
		Flex: tview.NewFlex(),
		view: view,
		cfg:  cfg,
	}

	p.Box = ui.ConfigureBox(tview.NewBox(), &cfg.Theme)

	// Create input field
	p.inputField = tview.NewInputField()
	p.inputField.SetLabel("> ")
	p.inputField.SetChangedFunc(p.onInputChanged)
	p.inputField.SetDoneFunc(p.onDone)
	p.inputField.SetInputCapture(p.onInputFieldCapture)

	// Create list for autocomplete suggestions
	p.list = tview.NewList()
	p.list.ShowSecondaryText(false)
	p.list.SetSelectedFunc(p.onListSelected)

	// Build layout: input field on top, list below
	p.Flex.
		SetDirection(tview.FlexRow).
		AddItem(p.inputField, 1, 0, true).
		AddItem(p.list, 0, 1, false)

	return p
}

type channelCandidate struct {
	name      string
	guildName string
	id        discord.ChannelID
}

func (c channelCandidate) String() string {
	if c.guildName != "" {
		return fmt.Sprintf("%s (%s)", c.name, c.guildName)
	}
	return c.name
}

func (p *picker) onInputChanged(text string) {
	p.updateAutocompleteList(text)
}

func (p *picker) updateAutocompleteList(currentText string) {
	if p.view.state == nil || p.view.state.Cabinet == nil {
		p.list.Clear()
		return
	}

	var candidates []channelCandidate

	// Guild Channels
	guilds, err := p.view.state.Cabinet.Guilds()
	if err != nil {
		return
	}
	for _, guild := range guilds {
		channels, err := p.view.state.Cabinet.Channels(guild.ID)
		if err != nil {
			continue
		}
		for _, ch := range channels {
			if ch.Type == discord.GuildText || ch.Type == discord.GuildNews || ch.Type == discord.GuildPublicThread || ch.Type == discord.GuildPrivateThread || ch.Type == discord.GuildAnnouncementThread {
				candidates = append(candidates, channelCandidate{
					name:      "#" + ch.Name,
					guildName: guild.Name,
					id:        ch.ID,
				})
			}
		}
	}

	// DM Channels
	privateChannels, err := p.view.state.PrivateChannels()
	if err != nil {
		return
	}
	for _, ch := range privateChannels {
		name := "Direct Message"
		if len(ch.DMRecipients) > 0 {
			name = ch.DMRecipients[0].Tag()
		}
		if ch.Name != "" {
			name = ch.Name
		}

		candidates = append(candidates, channelCandidate{
			name:      name,
			guildName: "Direct Messages",
			id:        ch.ID,
		})
	}

	matches := fuzzy.FindFrom(currentText, candidateList(candidates))
	sort.SliceStable(matches, func(i, j int) bool {
		return matches[i].Score > matches[j].Score
	})

	p.candidates = make([]channelCandidate, 0, len(matches))
	for _, match := range matches {
		candidate := candidates[match.Index]
		p.candidates = append(p.candidates, candidate)
	}

	if len(p.candidates) > 10 {
		p.candidates = p.candidates[:10]
	}

	// Update the list
	p.list.Clear()
	for _, candidate := range p.candidates {
		text := candidate.String()
		p.list.AddItem(text, "", 0, nil)
	}

	// Reset list selection to first item when candidates are updated
	if len(p.candidates) > 0 {
		p.list.SetCurrentItem(0)
	}
}

type candidateList []channelCandidate

func (cl candidateList) String(i int) string {
	return cl[i].String()
}

func (cl candidateList) Len() int {
	return len(cl)
}

func (p *picker) onListSelected(index int, mainText, secondaryText string, shortcut rune) {
	if index >= 0 && index < len(p.candidates) {
		candidate := p.candidates[index]
		p.view.guildsTree.SelectChannelID(candidate.id)
		p.view.togglePicker()
	}
}

func (p *picker) onInputFieldCapture(event *tcell.EventKey) *tcell.EventKey {
	if len(p.candidates) > 0 {
		switch event.Name() {
		case p.cfg.Keys.Picker.Up:
			p.list.InputHandler()(tcell.NewEventKey(tcell.KeyUp, "", tcell.ModNone), nil)
			return nil
		case p.cfg.Keys.Picker.Down:
			p.list.InputHandler()(tcell.NewEventKey(tcell.KeyDown, "", tcell.ModNone), nil)
			return nil
		case p.cfg.Keys.Picker.Confirm:
			index := p.list.GetCurrentItem()
			if index >= 0 && index < len(p.candidates) {
				candidate := p.candidates[index]
				p.view.guildsTree.SelectChannelID(candidate.id)
				p.view.togglePicker()
			}
			return nil
		}
	}

	switch event.Name() {
	case p.cfg.Keys.Picker.Cancel:
		p.view.togglePicker()
		return nil
	}
	return event
}

func (p *picker) onDone(key tcell.Key) {
	switch key {
	case tcell.KeyEnter:
		// If there are candidates, select the first one
		if len(p.candidates) > 0 {
			candidate := p.candidates[0]
			p.view.guildsTree.SelectChannelID(candidate.id)
			p.view.togglePicker()
		}
	}
}
