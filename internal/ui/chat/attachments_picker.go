package chat

import (
	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/discordo/internal/ui"
	"github.com/ayn2op/discordo/pkg/picker"
	"github.com/ayn2op/tview"
	"github.com/ayn2op/tview/help"
	"github.com/ayn2op/tview/keybind"
)

type attachmentItem struct {
	label string
	open  func()
}

type attachmentsPicker struct {
	*picker.Picker
	cfg      *config.Config
	chatView *View
	items    []attachmentItem
}

var _ help.KeyMap = (*attachmentsPicker)(nil)

func newAttachmentsPicker(cfg *config.Config, chatView *View) *attachmentsPicker {
	ap := &attachmentsPicker{
		Picker:   picker.New(),
		cfg:      cfg,
		chatView: chatView,
	}
	ap.Box = ui.ConfigureBox(tview.NewBox(), &cfg.Theme)
	ap.
		SetBlurFunc(nil).
		SetFocusFunc(nil).
		SetBorderSet(cfg.Theme.Border.ActiveSet.BorderSet).
		SetBorderStyle(cfg.Theme.Border.ActiveStyle.Style).
		SetTitleStyle(cfg.Theme.Title.ActiveStyle.Style).
		SetFooterStyle(cfg.Theme.Footer.ActiveStyle.Style)

	ap.SetTitle("Attachments")
	ap.SetSelectedFunc(ap.onSelected)
	ap.SetCancelFunc(ap.close)
	ap.SetKeyMap(&picker.KeyMap{
		Cancel: cfg.Keybinds.Picker.Cancel.Keybind,
		Up:     cfg.Keybinds.Picker.Up.Keybind,
		Down:   cfg.Keybinds.Picker.Down.Keybind,
		Top:    cfg.Keybinds.Picker.Top.Keybind,
		Bottom: cfg.Keybinds.Picker.Bottom.Keybind,
		Select: cfg.Keybinds.Picker.Select.Keybind,
	})
	ap.SetScrollBarVisibility(cfg.Theme.ScrollBar.Visibility.ScrollBarVisibility)
	ap.SetScrollBar(tview.NewScrollBar().
		SetTrackStyle(cfg.Theme.ScrollBar.TrackStyle.Style).
		SetThumbStyle(cfg.Theme.ScrollBar.ThumbStyle.Style).
		SetGlyphSet(cfg.Theme.ScrollBar.GlyphSet.GlyphSet))
	return ap
}

func (ap *attachmentsPicker) SetItems(items []attachmentItem) {
	ap.items = items
	ap.ClearItems()
	for i, item := range items {
		ap.AddItem(picker.Item{
			Text:       item.label,
			FilterText: item.label,
			Reference:  i,
		})
	}
	ap.Update()
}

func (ap *attachmentsPicker) onSelected(item picker.Item) {
	index, ok := item.Reference.(int)
	if !ok {
		return
	}
	if index < 0 || index >= len(ap.items) {
		return
	}
	ap.items[index].open()
	ap.close()
}

func (ap *attachmentsPicker) close() {
	ap.chatView.RemoveLayer(attachmentsListLayerName)
	ap.chatView.app.SetFocus(ap.chatView.messagesList)
}

func (ap *attachmentsPicker) ShortHelp() []keybind.Keybind {
	cfg := ap.cfg.Keybinds.Picker
	return []keybind.Keybind{cfg.Up.Keybind, cfg.Down.Keybind, cfg.Select.Keybind, cfg.Cancel.Keybind}
}

func (ap *attachmentsPicker) FullHelp() [][]keybind.Keybind {
	cfg := ap.cfg.Keybinds.Picker
	return [][]keybind.Keybind{
		{cfg.Up.Keybind, cfg.Down.Keybind, cfg.Top.Keybind, cfg.Bottom.Keybind},
		{cfg.Select.Keybind, cfg.Cancel.Keybind},
	}
}
