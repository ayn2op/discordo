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

	// Create input field
	qs.inputField = tview.NewInputField()
	qs.inputField.SetFieldBackgroundColor(tview.Styles.PrimitiveBackgroundColor)
	qs.inputField.SetFieldTextColor(tview.Styles.PrimaryTextColor)
	qs.inputField.SetLabel("Jump to: ")
	qs.inputField.SetChangedFunc(qs.onInputChanged)
	qs.inputField.SetDoneFunc(qs.onDone)
	qs.inputField.SetInputCapture(qs.onInputFieldCapture)

	// Create list for autocomplete suggestions with border
	qs.list = tview.NewList()
	qs.list.Box = ui.ConfigureBox(qs.list.Box, &cfg.Theme)
	if cfg.Theme.Border.Enabled {
		qs.list.SetBorders(tview.BordersAll)
	} else {
		// Always show border for quick switcher list to distinguish it from chat
		border := cfg.Theme.Border
		normalBorderStyle := border.NormalStyle.Style
		normalBorderSet := border.NormalSet.BorderSet
		qs.list.Box.
			SetBorderStyle(normalBorderStyle).
			SetBorderSet(normalBorderSet).
			SetBorders(tview.BordersAll)
	}
	qs.list.ShowSecondaryText(false)
	qs.list.SetSelectedFunc(qs.onListSelected)
	qs.list.SetInputCapture(qs.onListInputCapture)

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
	guilds, _ := qs.view.state.Cabinet.Guilds()
	for _, guild := range guilds {
		channels, _ := qs.view.state.Cabinet.Channels(guild.ID)
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
	privateChannels, _ := qs.view.state.PrivateChannels()
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

	var candidateStrings []string
	for _, c := range candidates {
		candidateStrings = append(candidateStrings, c.String())
	}

	matches := fuzzy.Find(currentText, candidateStrings)
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

	// If there are suggestions, switch focus to list when arrow key is pressed
	if len(qs.candidates) > 0 {
		qs.list.SetCurrentItem(0)
	}
}

func (qs *quickSwitcher) onListSelected(index int, mainText, secondaryText string, shortcut rune) {
	if index >= 0 && index < len(qs.candidates) {
		candidate := qs.candidates[index]
		qs.view.guildsTree.SelectChannelID(candidate.id)
		qs.view.toggleQuickSwitcher()
	}
}

func (qs *quickSwitcher) onListInputCapture(event *tcell.EventKey) *tcell.EventKey {
	switch event.Key() {
	case tcell.KeyEnter:
		index := qs.list.GetCurrentItem()
		if index >= 0 && index < len(qs.candidates) {
			candidate := qs.candidates[index]
			qs.view.guildsTree.SelectChannelID(candidate.id)
			qs.view.toggleQuickSwitcher()
		}
		return nil
	case tcell.KeyEscape:
		qs.view.toggleQuickSwitcher()
		return nil
	case tcell.KeyUp:
		if qs.list.GetCurrentItem() == 0 {
			// Move focus back to input field
			qs.view.app.SetFocus(qs.inputField)
			return nil
		}
	case tcell.KeyTab:
		// Tab moves focus between input and list
		if qs.view.app.GetFocus() == qs.inputField {
			qs.view.app.SetFocus(qs.list)
			return nil
		} else if qs.view.app.GetFocus() == qs.list {
			qs.view.app.SetFocus(qs.inputField)
			return nil
		}
	}
	return event
}

func (qs *quickSwitcher) onInputFieldCapture(event *tcell.EventKey) *tcell.EventKey {
	switch event.Key() {
	case tcell.KeyDown:
		// Move focus to list if there are suggestions
		if len(qs.candidates) > 0 {
			qs.view.app.SetFocus(qs.list)
			return nil
		}
	case tcell.KeyTab:
		// Move focus to list if there are suggestions
		if len(qs.candidates) > 0 {
			qs.view.app.SetFocus(qs.list)
			return nil
		}
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
		} else {
			// Try to match exact text
			text := qs.inputField.GetText()
			for _, c := range qs.candidates {
				if c.String() == text {
					qs.view.guildsTree.SelectChannelID(c.id)
					qs.view.toggleQuickSwitcher()
					return
				}
			}
		}
	case tcell.KeyEscape:
		qs.view.toggleQuickSwitcher()
	}
}
