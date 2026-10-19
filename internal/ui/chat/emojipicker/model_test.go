package emojipicker

import (
	"path/filepath"
	"testing"

	"github.com/ayn2op/arikawa/v3/discord"
	"github.com/ayn2op/arikawa/v3/state"
	"github.com/ayn2op/arikawa/v3/state/store/defaultstore"
	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/ningen/v3"
	ningenEmoji "github.com/ayn2op/ningen/v3/states/emoji"
)

func TestEmojiPickerRefresh(t *testing.T) {
	cfg, err := config.Load(filepath.Join(t.TempDir(), "config.toml"))
	if err != nil {
		t.Fatal(err)
	}

	st := ningen.FromState(state.NewWithStore("", defaultstore.New()))
	guildID := discord.GuildID(100)
	st.EmojiState = ningenEmoji.NewState(st.Cabinet)
	_ = st.Cabinet.GuildSet(&discord.Guild{ID: guildID}, false)
	_ = st.Cabinet.EmojiSet(guildID, []discord.Emoji{
		{ID: 555, Name: "custom_pepe"},
	}, false)

	p := NewModel(cfg)
	p.RefreshEmojis(st, guildID)

	widget := p.View()
	if widget == nil {
		t.Fatal("expected non-nil view")
	}

	// Test Update
	p, _ = p.Update(CancelMsg{})
}
