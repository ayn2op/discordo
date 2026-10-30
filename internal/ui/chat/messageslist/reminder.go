package messageslist

import (
	"errors"
	"net/http"
	"strings"
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

// ShowReminderPromptMsg asks to type when to be reminded. Set turns what is typed into the command that sets the reminder, or into the error to show for a time it cannot read.
type ShowReminderPromptMsg struct {
	Set func(text string) (tview.Cmd, error)
}

// customLabel is the picker item that asks for a time instead of offering one.
const customLabel = "Custom..."

// timeLayouts are the formats a typed time is read in, after a duration is tried, all in local time.
var timeLayouts = []string{"15:04", "2006-01-02 15:04", "2006-01-02"}

var errPast = errors.New("time is in the past")

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

	items := make(picker.Items, 0, len(reminderPresets)+1)
	for _, preset := range reminderPresets {
		items = append(items, picker.Item{Text: preset.label, Reference: m.setReminder(*selectedMessage, preset.due)})
	}
	items = append(items, picker.Item{Text: customLabel, Reference: m.promptReminder(*selectedMessage)})
	return func() tview.Msg { return ShowRemindersMsg(items) }
}

// promptReminder asks to type when to be reminded of message.
func (m Model) promptReminder(message discord.Message) tview.Cmd {
	set := func(text string) (tview.Cmd, error) {
		due, err := parseReminderTime(text, time.Now())
		if err != nil {
			return nil, err
		}
		return m.setReminder(message, func(time.Time) time.Time { return due }), nil
	}
	return func() tview.Msg { return ShowReminderPromptMsg{Set: set} }
}

// parseReminderTime reads text as a delay from now ("45m", "2h30m") or a local time in one of timeLayouts.
// A clock time already past today is read as tomorrow's, and a date alone as its morning.
func parseReminderTime(text string, now time.Time) (time.Time, error) {
	text = strings.TrimSpace(text)
	if delay, err := time.ParseDuration(text); err == nil {
		if delay <= 0 {
			return time.Time{}, errPast
		}
		return now.Add(delay), nil
	}

	now = now.Local()
	for _, layout := range timeLayouts {
		parsed, err := time.ParseInLocation(layout, text, now.Location())
		if err != nil {
			continue
		}

		switch layout {
		case "15:04":
			parsed = time.Date(now.Year(), now.Month(), now.Day(), parsed.Hour(), parsed.Minute(), 0, 0, now.Location())
			if !parsed.After(now) {
				parsed = parsed.AddDate(0, 0, 1)
			}
		case "2006-01-02":
			parsed = time.Date(parsed.Year(), parsed.Month(), parsed.Day(), morningHour, 0, 0, 0, now.Location())
		}
		if !parsed.After(now) {
			return time.Time{}, errPast
		}
		return parsed, nil
	}

	return time.Time{}, errors.New("expected a delay like 45m or a time like " + strings.Join(timeLayouts, ", "))
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
