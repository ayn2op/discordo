package messageslist

import (
	"testing"
	"time"

	"github.com/ayn2op/arikawa/v3/discord"
)

func TestModelRemind(t *testing.T) {
	t.Run("no selection", func(t *testing.T) {
		ml := newTestModel(t, testConfig(t))
		ml.setMessages([]discord.Message{{ID: 2}})
		if ml.remind() != nil {
			t.Fatal("remind without a selected message returned a command")
		}
	})

	t.Run("selected", func(t *testing.T) {
		ml := newTestModel(t, testConfig(t))
		ml.setMessages([]discord.Message{{ID: 2}})
		ml.selectBottom()
		cmd := ml.remind()
		if cmd == nil {
			t.Fatal("remind with a selected message returned no command")
		}
		items, ok := cmd().(ShowRemindersMsg)
		if !ok {
			t.Fatalf("remind returned %T, want ShowRemindersMsg", cmd())
		}
		if len(items) != len(reminderPresets)+1 {
			t.Fatalf("picker has %d items, want %d", len(items), len(reminderPresets)+1)
		}
		for i, preset := range reminderPresets {
			if items[i].Text != preset.label || items[i].Reference == nil {
				t.Fatalf("item %d is %q with action %v", i, items[i].Text, items[i].Reference)
			}
		}
		if custom := items[len(items)-1]; custom.Text != customLabel || custom.Reference == nil {
			t.Fatalf("last item is %q with action %v", custom.Text, custom.Reference)
		}
	})
}

func TestNextMonday(t *testing.T) {
	// 2026-10-28 is a Wednesday, 2026-10-26 a Monday.
	wednesday := time.Date(2026, time.October, 28, 15, 30, 0, 0, time.Local)
	monday := time.Date(2026, time.October, 26, 15, 30, 0, 0, time.Local)
	for name, tc := range map[string]struct{ now, want time.Time }{
		"midweek": {wednesday, time.Date(2026, time.November, 2, morningHour, 0, 0, 0, time.Local)},
		"monday":  {monday, time.Date(2026, time.November, 2, morningHour, 0, 0, 0, time.Local)},
	} {
		t.Run(name, func(t *testing.T) {
			if got := nextMonday(tc.now); !got.Equal(tc.want) {
				t.Fatalf("nextMonday(%v) = %v, want %v", tc.now, got, tc.want)
			}
		})
	}
}

func TestMorningOf(t *testing.T) {
	now := time.Date(2026, time.October, 31, 23, 59, 0, 0, time.Local)
	want := time.Date(2026, time.November, 1, morningHour, 0, 0, 0, time.Local)
	if got := morningOf(now, 1); !got.Equal(want) {
		t.Fatalf("morningOf(%v, 1) = %v, want %v", now, got, want)
	}
}

func TestParseReminderTime(t *testing.T) {
	// 2026-10-28 is a Wednesday.
	now := time.Date(2026, time.October, 28, 15, 30, 0, 0, time.Local)
	for name, tc := range map[string]struct {
		text string
		want time.Time
		err  bool
	}{
		"duration":        {text: "2h30m", want: now.Add(2*time.Hour + 30*time.Minute)},
		"padded duration": {text: " 45m ", want: now.Add(45 * time.Minute)},
		"zero duration":   {text: "0s", err: true},
		"clock later":     {text: "18:30", want: time.Date(2026, time.October, 28, 18, 30, 0, 0, time.Local)},
		"clock passed":    {text: "09:00", want: time.Date(2026, time.October, 29, 9, 0, 0, 0, time.Local)},
		"date and time":   {text: "2026-11-02 09:15", want: time.Date(2026, time.November, 2, 9, 15, 0, 0, time.Local)},
		"date":            {text: "2026-11-02", want: time.Date(2026, time.November, 2, morningHour, 0, 0, 0, time.Local)},
		"date passed":     {text: "2026-10-27", err: true},
		"garbage":         {text: "next tuesday", err: true},
		"empty":           {text: "", err: true},
	} {
		t.Run(name, func(t *testing.T) {
			got, err := parseReminderTime(tc.text, now)
			if (err != nil) != tc.err {
				t.Fatalf("parseReminderTime(%q) error = %v, want error %t", tc.text, err, tc.err)
			}
			if !tc.err && !got.Equal(tc.want) {
				t.Fatalf("parseReminderTime(%q) = %v, want %v", tc.text, got, tc.want)
			}
		})
	}
}
