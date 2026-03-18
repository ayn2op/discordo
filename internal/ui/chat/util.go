package chat

import (
	"strings"

	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/discordo/internal/ui"
	"github.com/ayn2op/tview"
	"github.com/ayn2op/tview/list"
	"github.com/ayn2op/tview/picker"
)

func ConfigurePicker(model *picker.Model, cfg *config.Config, title string) {
	model.Box = ui.ConfigureBox(tview.NewBox(), &cfg.Theme)
	// When a child of the parent flex is focused, the parent layout itself is not reported as focused.
	// Instead, the focused child (picker) is considered focused.
	// Therefore, we manually set the active border style on the picker to ensure it displays the correct focused appearance.
	model.
		SetBlurFunc(nil).
		SetFocusFunc(nil).
		SetBorderSet(cfg.Theme.Border.ActiveSet.BorderSet).
		SetBorderStyle(cfg.Theme.Border.ActiveStyle.Style).
		SetTitleStyle(cfg.Theme.Title.ActiveStyle.Style).
		SetFooterStyle(cfg.Theme.Footer.ActiveStyle.Style)

	model.SetTitle(title)
	model.SetScrollBarVisibility(cfg.Theme.ScrollBar.Visibility.ScrollBarVisibility)
	model.SetScrollBar(tview.NewScrollBar().
		SetTrackStyle(cfg.Theme.ScrollBar.TrackStyle.Style).
		SetThumbStyle(cfg.Theme.ScrollBar.ThumbStyle.Style).
		SetGlyphSet(cfg.Theme.ScrollBar.GlyphSet.GlyphSet))
	model.SetKeybinds(picker.Keybinds{
		Cancel: cfg.Keybinds.Picker.Cancel.Keybind,
		Keybinds: list.Keybinds{
			SelectUp:     cfg.Keybinds.Picker.Up.Keybind,
			SelectDown:   cfg.Keybinds.Picker.Down.Keybind,
			SelectTop:    cfg.Keybinds.Picker.Top.Keybind,
			SelectBottom: cfg.Keybinds.Picker.Bottom.Keybind,
		},
		Select: cfg.Keybinds.Picker.Select.Keybind,
	})
}

func humanJoin(items []string) string {
	count := len(items)
	switch count {
	case 0:
		return ""
	case 1:
		return items[0]
	case 2:
		return items[0] + " and " + items[1]
	default:
		return strings.Join(items[:count-1], ", ") + ", and " + items[count-1]
	}
}
