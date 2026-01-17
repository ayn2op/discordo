package chat

import (
	"fmt"
	"sort"

	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/tview"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/gdamore/tcell/v3"
	"github.com/sahilm/fuzzy"
)

type quickSwitcher struct {
	*tview.InputField
	view *View
	cfg  *config.Config

	candidates []channelCandidate
}

func newQuickSwitcher(view *View, cfg *config.Config) *quickSwitcher {
	qs := &quickSwitcher{
		InputField: tview.NewInputField(),
		view:       view,
		cfg:        cfg,
	}

	qs.SetFieldBackgroundColor(tview.Styles.PrimitiveBackgroundColor)
	qs.SetFieldTextColor(tview.Styles.PrimaryTextColor)
	qs.SetLabel("Jump to: ")
	qs.SetAutocompleteFunc(qs.autocomplete)
	qs.SetAutocompletedFunc(qs.onAutocompleted)
	qs.SetDoneFunc(qs.onDone)

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

func (qs *quickSwitcher) autocomplete(currentText string) []string {
	if qs.view.state == nil || qs.view.state.Cabinet == nil {
		return nil
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
	var suggestions []string
	for _, match := range matches {
		candidate := candidates[match.Index]
		qs.candidates = append(qs.candidates, candidate)
		suggestions = append(suggestions, candidate.String())
	}

	if len(suggestions) > 10 {
		suggestions = suggestions[:10]
		qs.candidates = qs.candidates[:10]
	}

	return suggestions
}

func (qs *quickSwitcher) onAutocompleted(text string, index int, source int) bool {
	for _, c := range qs.candidates {
		if c.String() == text {
			qs.view.guildsTree.SelectChannelID(c.id)
			qs.view.toggleQuickSwitcher()
			return true
		}
	}
	return true
}

func (qs *quickSwitcher) onDone(key tcell.Key) {
	if key == tcell.KeyEnter {
		text := qs.GetText()
		for _, c := range qs.candidates {
			if c.String() == text {
				qs.view.guildsTree.SelectChannelID(c.id)
				qs.view.toggleQuickSwitcher()
				return
			}
		}
	} else if key == tcell.KeyEscape {
		qs.view.toggleQuickSwitcher()
	}
}
