package messageslist

import (
	"testing"

	"github.com/ayn2op/arikawa/v3/discord"
)

func TestModelEmbedViewsDisabled(t *testing.T) {
	ml := newTestModel(t, testConfig(t))
	message := discord.Message{
		Type:   discord.DefaultMessage,
		Embeds: []discord.Embed{{Title: "title"}},
	}

	if views := ml.embedViews(message, nil); len(views) == 0 {
		t.Fatal("embedViews() = nil, want a view while embeds are enabled")
	}

	ml.cfg.UI.MessagesList.Embeds.Enabled = false
	if views := ml.embedViews(message, nil); views != nil {
		t.Fatalf("embedViews() = %v, want nil while embeds are disabled", views)
	}
}
