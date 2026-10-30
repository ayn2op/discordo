package messageslist

import (
	"testing"

	"github.com/ayn2op/arikawa/v3/discord"
	"github.com/ayn2op/discordo/internal/ui"
)

func TestModelUpdateConfirmDelete(t *testing.T) {
	// selected is mine throughout, so a confirmation that went by the selection would always ask.
	selected := discord.Message{ID: 2, ChannelID: 1, Author: discord.User{ID: 1}}
	for _, tt := range []struct {
		name    string
		message discord.Message
		want    bool
	}{
		{"my own message", discord.Message{ID: 3, ChannelID: 1, Author: discord.User{ID: 1}}, true},
		{"someone else's message", discord.Message{ID: 4, ChannelID: 1, Author: discord.User{ID: 2}}, false},
	} {
		t.Run("asked to delete "+tt.name, func(t *testing.T) {
			ml := newTestModel(t, testConfig(t))
			ml.setMessages([]discord.Message{selected, tt.message})
			ml.selectBottom()

			var asked bool
			if _, cmd := ml.Update(ConfirmDeleteMsg(tt.message)); cmd != nil {
				_, asked = cmd().(ui.DialogMsg)
			}

			if asked != tt.want {
				t.Fatalf("asked to confirm = %v, want %v", asked, tt.want)
			}
		})
	}
}
