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

// reminderPreset is a choice the reminder picker offers, due at what due returns for the time it is picked at.
type reminderPreset struct {
	label string
	due   func(now time.Time) time.Time
}

// morningHour is the hour, in local time, that day-based reminders are due at.
const morningHour = 9

// reminderPresets are the choices the Discord client offers, less its custom date.
var reminderPresets = []reminderPreset{
	{"In 30 minutes", after(30 * time.Minute)},
	{"In 1 hour", after(time.Hour)},
	{"In 4 hours", after(4 * time.Hour)},
	{"Tomorrow at 09:00", func(now time.Time) time.Time { return morningOf(now, 1) }},
	{"Monday at 09:00", nextMonday},
}

func after(delay time.Duration) func(time.Time) time.Time {
	return func(now time.Time) time.Time { return now.Add(delay) }
}

// morningOf returns morningHour, local time, days after now.
func morningOf(now time.Time, days int) time.Time {
	now = now.Local()
	return time.Date(now.Year(), now.Month(), now.Day()+days, morningHour, 0, 0, 0, now.Location())
}

// nextMonday returns the morning of the Monday after now, a week ahead if now is a Monday.
func nextMonday(now time.Time) time.Time {
	days := (int(time.Monday) - int(now.Local().Weekday()) + 7) % 7
	if days == 0 {
		days = 7
	}
	return morningOf(now, days)
}

// remind shows when to be reminded of the selected message, each choice setting that reminder.
func (m Model) remind() tview.Cmd {
	selectedMessage, ok := m.selectedMessage()
	if !ok {
		return nil
	}

	items := make(picker.Items, len(reminderPresets))
	for i, preset := range reminderPresets {
		items[i] = picker.Item{Text: preset.label, Reference: m.setReminder(*selectedMessage, preset.due)}
	}
	return func() tview.Msg { return ShowRemindersMsg(items) }
}

// setReminder saves message with a reminder due at what due returns for when it runs.
func (m Model) setReminder(message discord.Message, due func(time.Time) time.Time) tview.Cmd {
	return func() tview.Msg {
		if err := m.saveMessage(message, due(time.Now())); err != nil {
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
