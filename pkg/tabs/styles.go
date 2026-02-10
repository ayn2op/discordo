package tabs

import "charm.land/lipgloss/v2"

type Styles struct {
	TabStyle       lipgloss.Style
	ActiveTabStyle lipgloss.Style
	TabLineStyle   lipgloss.Style
}

func DefaultStyles() Styles {
	tabStyle := lipgloss.NewStyle().Padding(0, 1).Align(lipgloss.Center)
	return Styles{
		TabStyle:       tabStyle,
		ActiveTabStyle: tabStyle.Reverse(true),
		TabLineStyle:   lipgloss.NewStyle().Align(lipgloss.Center, lipgloss.Center),
	}
}
