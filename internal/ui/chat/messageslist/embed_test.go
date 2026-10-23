package messageslist

import (
	"testing"

	"github.com/ayn2op/arikawa/v3/discord"
)

func TestModelEmbedViews(t *testing.T) {
	ml := newTestModel(t, testConfig(t))
	message := discord.Message{
		Type:   discord.DefaultMessage,
		Embeds: []discord.Embed{{Title: "title"}},
	}

	t.Run("enabled", func(t *testing.T) {
		if views := ml.embedViews(message, nil); len(views) == 0 {
			t.Fatal("embedViews() = nil, want a view while embeds are enabled")
		}
	})

	t.Run("disabled", func(t *testing.T) {
		ml.cfg.UI.MessagesList.Embeds.Enabled = false
		if views := ml.embedViews(message, nil); views != nil {
			t.Fatalf("embedViews() = %v, want nil while embeds are disabled", views)
		}
	})
}
