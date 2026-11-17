package messageslist

import (
	"testing"

	"github.com/ayn2op/arikawa/v3/discord"
)

func TestModelUpdateDelete(t *testing.T) {
	// selected is mine throughout, so a deletion that went by the selection would always go ahead.
	selected := discord.Message{ID: 2, ChannelID: 1, Author: discord.User{ID: 1}}
	for _, tt := range []struct {
		name    string
		message discord.Message
		want    bool
	}{
		{"my own message", discord.Message{ID: 3, ChannelID: 1, Author: discord.User{ID: 1}}, true},
		{"someone else's message", discord.Message{ID: 4, ChannelID: 1, Author: discord.User{ID: 2}}, false},
	} {
		t.Run("deleting "+tt.name, func(t *testing.T) {
			ml := newTestModel(t, testConfig(t))
			ml.setMessages([]discord.Message{selected, tt.message})
			ml.selectBottom()

			// The request itself is left unrun, as it would reach Discord; that it was made is the ask.
			_, cmd := ml.Update(DeleteMsg(tt.message))
			if requested := cmd != nil; requested != tt.want {
				t.Fatalf("requested the deletion = %v, want %v", requested, tt.want)
			}
		})
	}
}
