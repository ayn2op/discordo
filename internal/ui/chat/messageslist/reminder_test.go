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
		if len(items) != len(reminderPresets) {
			t.Fatalf("picker has %d items, want %d", len(items), len(reminderPresets))
		}
		for i, item := range items {
			if item.Text != reminderPresets[i].label || item.Reference == nil {
				t.Fatalf("item %d is %q with action %v", i, item.Text, item.Reference)
			}
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
