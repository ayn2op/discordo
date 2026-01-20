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

type quickSwitcher struct {
	*tview.Flex
	view *View
	cfg  *config.Config

	inputField *tview.InputField
	list       *tview.List

	candidates []channelCandidate
}

func newQuickSwitcher(view *View, cfg *config.Config) *quickSwitcher {
	qs := &quickSwitcher{
		Flex: tview.NewFlex(),
		view: view,
		cfg:  cfg,
	}

	qs.Box = ui.ConfigureBox(tview.NewBox(), &cfg.Theme)

	// Create input field
	qs.inputField = tview.NewInputField()
	qs.inputField.SetLabel("> ")
	qs.inputField.SetChangedFunc(qs.onInputChanged)
	qs.inputField.SetDoneFunc(qs.onDone)
	qs.inputField.SetInputCapture(qs.onInputFieldCapture)

	// Create list for autocomplete suggestions
	qs.list = tview.NewList()
	qs.list.ShowSecondaryText(false)
	qs.list.SetSelectedFunc(qs.onListSelected)

	// Build layout: input field on top, list below
	qs.Flex.
		SetDirection(tview.FlexRow).
		AddItem(qs.inputField, 1, 0, true).
		AddItem(qs.list, 0, 1, false)

	return qs
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

func (qs *quickSwitcher) onInputChanged(text string) {
	qs.updateAutocompleteList(text)
}

func (qs *quickSwitcher) updateAutocompleteList(currentText string) {
	if qs.view.state == nil || qs.view.state.Cabinet == nil {
		qs.list.Clear()
		return
	}

	var candidates []channelCandidate

	// Guild Channels
	guilds, err := qs.view.state.Cabinet.Guilds()
	if err != nil {
		return
	}
	for _, guild := range guilds {
		channels, err := qs.view.state.Cabinet.Channels(guild.ID)
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
	privateChannels, err := qs.view.state.PrivateChannels()
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

	qs.candidates = make([]channelCandidate, 0, len(matches))
	for _, match := range matches {
		candidate := candidates[match.Index]
		qs.candidates = append(qs.candidates, candidate)
	}

	if len(qs.candidates) > 10 {
		qs.candidates = qs.candidates[:10]
	}

	// Update the list
	qs.list.Clear()
	for _, candidate := range qs.candidates {
		text := candidate.String()
		qs.list.AddItem(text, "", 0, nil)
	}

	// Reset list selection to first item when candidates are updated
	if len(qs.candidates) > 0 {
		qs.list.SetCurrentItem(0)
	}
}

type candidateList []channelCandidate

func (cl candidateList) String(i int) string {
	return cl[i].String()
}

func (cl candidateList) Len() int {
	return len(cl)
}

func (qs *quickSwitcher) onListSelected(index int, mainText, secondaryText string, shortcut rune) {
	if index >= 0 && index < len(qs.candidates) {
		candidate := qs.candidates[index]
		qs.view.guildsTree.SelectChannelID(candidate.id)
		qs.view.toggleQuickSwitcher()
	}
}

func (qs *quickSwitcher) onInputFieldCapture(event *tcell.EventKey) *tcell.EventKey {
	if len(qs.candidates) > 0 {
		switch event.Name() {
		case qs.cfg.Keys.Picker.Up:
			qs.list.InputHandler()(tcell.NewEventKey(tcell.KeyUp, "", tcell.ModNone), nil)
			return nil
		case qs.cfg.Keys.Picker.Down:
			qs.list.InputHandler()(tcell.NewEventKey(tcell.KeyDown, "", tcell.ModNone), nil)
			return nil
		case qs.cfg.Keys.Picker.Confirm:
			index := qs.list.GetCurrentItem()
			if index >= 0 && index < len(qs.candidates) {
				candidate := qs.candidates[index]
				qs.view.guildsTree.SelectChannelID(candidate.id)
				qs.view.toggleQuickSwitcher()
			}
			return nil
		}
	}

	switch event.Name() {
	case qs.cfg.Keys.Picker.Cancel:
		qs.view.toggleQuickSwitcher()
		return nil
	}
	return event
}

func (qs *quickSwitcher) onDone(key tcell.Key) {
	switch key {
	case tcell.KeyEnter:
		// If there are candidates, select the first one
		if len(qs.candidates) > 0 {
			candidate := qs.candidates[0]
			qs.view.guildsTree.SelectChannelID(candidate.id)
			qs.view.toggleQuickSwitcher()
		}
	}
}
