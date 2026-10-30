package messageslist

import (
	"testing"

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
