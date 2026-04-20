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
	cfg   *config.Config
	chat  *Model
	items []attachmentItem
}

var _ help.KeyMap = (*attachmentsPicker)(nil)

func newAttachmentsPicker(cfg *config.Config, chat *Model) *attachmentsPicker {
	ap := &attachmentsPicker{Model: picker.NewModel(), cfg: cfg, chat: chat}
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

func (ap *attachmentsPicker) close() tview.Cmd {

	ap.chat.RemoveLayer(attachmentsPickerLayerName)
	return tview.SetFocus(ap.chat.messagesList)
}

func (ap *attachmentsPicker) Update(msg tview.Msg) tview.Cmd {
	switch msg := msg.(type) {
	case picker.SelectedMsg:
		index, ok := msg.Reference.(int)
		if !ok {
			return nil
		}
		if index < 0 || index >= len(ap.items) {
			return nil
		}
		ap.items[index].open()
		return ap.close()
	case picker.CancelMsg:
		return ap.close()
	}

	return ap.Model.Update(msg)
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
