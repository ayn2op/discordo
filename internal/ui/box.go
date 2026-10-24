package ui

import (
	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/tview"
	"github.com/ayn2op/tview/box"
	"github.com/ayn2op/tview/text"
)

// Box returns child in a box laid out and styled by cfg, highlighted while focused.
func Box(child tview.Widget, cfg *config.Config, focused bool) box.Widget {
	ui, theme := cfg.UI, cfg.Theme
	padding := ui.Border.Padding
	widget := box.New(child).
		Padding(padding[0], padding[1], padding[2], padding[3]).
		TitleAlignment(ui.Title.Alignment.HorizontalAlignment).
		FooterAlignment(ui.Footer.Alignment.HorizontalAlignment)
	if ui.Border.Enabled {
		widget = widget.Borders(tview.BordersAll)
	}

	if focused {
		return widget.BorderStyle(theme.Border.ActiveStyle.Style).BorderSet(ui.Border.ActiveSet.BorderSet)
	}

	return widget.BorderStyle(theme.Border.NormalStyle.Style).BorderSet(ui.Border.NormalSet.BorderSet)
}

// Label returns label on one line in the style of theme for whether its pane is focused, as the title or footer of a box, or nil if label is empty, so the box keeps no row for it.
func Label(label string, theme config.StyleThemeConfig, focused bool) tview.Widget {
	if label == "" {
		return nil
	}

	style := theme.NormalStyle.Style
	if focused {
		style = theme.ActiveStyle.Style
	}

	return text.New(label).Style(style).Wrapping(text.WrappingNone)
}
