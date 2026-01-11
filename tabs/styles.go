package tabs

import "charm.land/lipgloss/v2"

type Styles struct {
	InactiveTab lipgloss.Style
	ActiveTab   lipgloss.Style
}

func DefaultStyles() Styles {
	inactiveTabStyle := lipgloss.NewStyle().Padding(0, 1)
	return Styles{
		InactiveTab: inactiveTabStyle,
		ActiveTab:   inactiveTabStyle.Background(lipgloss.Blue),
	}
}
