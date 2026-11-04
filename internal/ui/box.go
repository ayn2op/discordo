package ui

import (
	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/tview"
	"github.com/ayn2op/tview/box"
	"github.com/ayn2op/tview/text"
	"github.com/gdamore/tcell/v3"
	"github.com/gdamore/tcell/v3/color"
)

// Box returns child in a box laid out and styled by cfg, highlighted while focused.
func Box(child tview.Widget, cfg *config.Config, focused bool) box.Widget {
	ui := cfg.UI
	padding := ui.Border.Padding
	widget := box.New(child).
		Padding(padding[0], padding[1], padding[2], padding[3]).
		TitleAlignment(ui.Title.Alignment.HorizontalAlignment).
		FooterAlignment(ui.Footer.Alignment.HorizontalAlignment)
	if ui.Border.Enabled {
		widget = widget.Borders(tview.BordersAll)
	}

	set := ui.Border.NormalSet.BorderSet
	if focused {
		set = ui.Border.ActiveSet.BorderSet
	}

	return widget.BorderStyle(FocusStyle(cfg.Palette(), focused)).BorderSet(set)
}

// Label returns label on one line, highlighted while its pane is focused, as the title or footer of a box, or nil if label is empty, so the box keeps no row for it.
func Label(label string, cfg *config.Config, focused bool) tview.Widget {
	if label == "" {
		return nil
	}

	return text.New(label).Style(FocusStyle(cfg.Palette(), focused)).Wrapping(text.WrappingNone)
}

// FocusStyle is the strong primary color while focused, and the weak secondary one otherwise.
func FocusStyle(palette config.ThemeConfig, focused bool) tview.Style {
	if focused {
		return Foreground(palette.Primary.Strong.Style)
	}
	return Foreground(palette.Secondary.Weak.Style)
}

// SelectedStyle is the style of a selected item.
func SelectedStyle(palette config.ThemeConfig) tview.Style {
	return palette.Background.Neutral.Style
}

// MutedStyle is the style of what is drawn quietly, such as scroll bars.
func MutedStyle(palette config.ThemeConfig) tview.Style {
	return Foreground(palette.Secondary.Base.Style)
}

// ButtonStyle is the style of a button while focused, hovered, or neither.
func ButtonStyle(palette config.ThemeConfig, focused, hovered bool) tview.Style {
	switch {
	case focused:
		return palette.Primary.Base.Style
	case hovered:
		return palette.Background.Weak.Style
	}
	return palette.Background.Base.Style
}

// Foreground returns the background color of style as a foreground, for a palette color on text or lines. A style without a background color is returned as is.
func Foreground(style tview.Style) tview.Style {
	if bg := style.GetBackground(); bg != color.Default {
		return tcell.StyleDefault.Foreground(bg)
	}
	return style
}
