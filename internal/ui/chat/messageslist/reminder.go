package messageslist

import (
	"net/http"
	"time"

	"github.com/ayn2op/arikawa/v3/api"
	"github.com/ayn2op/arikawa/v3/discord"
	"github.com/ayn2op/arikawa/v3/utils/httputil"
	"github.com/ayn2op/discordo/internal/ui"
	"github.com/ayn2op/tview"
	"github.com/ayn2op/tview/picker"
)

// ShowRemindersMsg asks to pick when to be reminded of the selected message.
type ShowRemindersMsg picker.Items

// reminderPreset is a delay the reminder picker offers.
type reminderPreset struct {
	label string
	delay time.Duration
}

// reminderPresets are the delays the picker offers.
var reminderPresets = []reminderPreset{
	{"In 20 minutes", 20 * time.Minute},
	{"In 1 hour", time.Hour},
	{"In 3 hours", 3 * time.Hour},
	{"In 1 day", 24 * time.Hour},
	{"In 1 week", 7 * 24 * time.Hour},
}

// remind shows the delays to pick from, each setting a reminder for the selected message.
func (m Model) remind() tview.Cmd {
	selectedMessage, ok := m.selectedMessage()
	if !ok {
		return nil
	}

	items := make(picker.Items, len(reminderPresets))
	for i, preset := range reminderPresets {
		items[i] = picker.Item{Text: preset.label, Reference: m.setReminder(*selectedMessage, preset.delay)}
	}
	return func() tview.Msg { return ShowRemindersMsg(items) }
}

// setReminder saves message with a reminder due after delay from when it runs.
func (m Model) setReminder(message discord.Message, delay time.Duration) tview.Cmd {
	return func() tview.Msg {
		if err := m.saveMessage(message, time.Now().Add(delay)); err != nil {
			return ui.ErrorDialog("set reminder", err)
		}
		return nil
	}
}

// saveMessage bookmarks message, due as a reminder at dueAt.
// Discord's client calls this endpoint too; it is not in the bot API.
func (m Model) saveMessage(message discord.Message, dueAt time.Time) error {
	endpoint := api.EndpointMe + "/saved-messages/" + message.ChannelID.String() + "/" + message.ID.String()
	body := struct {
		DueAt time.Time `json:"due_at"`
	}{dueAt.UTC()}
	return m.state.FastRequest(http.MethodPut, endpoint, httputil.WithJSONBody(body))
}
