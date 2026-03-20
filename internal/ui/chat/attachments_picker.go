package chat

import (
	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/tview"
	"github.com/ayn2op/tview/help"
	"github.com/ayn2op/tview/keybind"
	"github.com/ayn2op/tview/picker"
)

type attachmentItem struct {
	label string
	open  func()
}

type attachmentsPicker struct {
	*picker.Model
	cfg      *config.Config
	chatView *Model
	items    []attachmentItem
}

var _ help.KeyMap = (*attachmentsPicker)(nil)

func newAttachmentsPicker(cfg *config.Config, chatView *Model) *attachmentsPicker {
	ap := &attachmentsPicker{
		Model:    picker.NewModel(),
		cfg:      cfg,
		chatView: chatView,
	}
	ConfigurePicker(ap.Model, cfg, "Attachments")
	return ap
}

func (ap *attachmentsPicker) SetItems(items []attachmentItem) {
	ap.items = items
	pickerItems := make(picker.Items, 0, len(items))
	for i, item := range items {
		pickerItems = append(pickerItems, picker.Item{
			Text:       item.label,
			FilterText: item.label,
			Reference:  i,
		})
	}
	ap.Model.SetItems(pickerItems)
}

func (ap *attachmentsPicker) close() {
	ap.chatView.RemoveLayer(attachmentsPickerLayerName)
	ap.chatView.app.SetFocus(ap.chatView.messagesList)
}

func (ap *attachmentsPicker) HandleEvent(event tview.Event) tview.Command {
	switch event := event.(type) {
	case *picker.SelectedEvent:
		index, ok := event.Reference.(int)
		if !ok {
			return nil
		}
		if index < 0 || index >= len(ap.items) {
			return nil
		}
		ap.items[index].open()
		ap.close()
		return nil
	case *picker.CancelEvent:
		ap.close()
		return nil
	}

	return ap.Model.HandleEvent(event)
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
