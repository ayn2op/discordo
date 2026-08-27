package root

import (
	"path/filepath"
	"testing"

	"github.com/ayn2op/discordo/internal/config"
	"github.com/gdamore/tcell/v3"
)

func TestHelpEnabled(t *testing.T) {
	cfg, err := config.Load(filepath.Join(t.TempDir(), "missing.toml"))
	if err != nil {
		t.Fatal(err)
	}

	t.Run("enabled", func(t *testing.T) {
		cfg := *cfg
		cfg.Help.Enabled = true
		m := NewModel(&cfg)
		if m.help == nil {
			t.Fatal("help model was not created")
		}
		if count := m.rootFlex.GetItemCount(); count != 1 {
			t.Fatalf("layout item count = %d, want 1", count)
		}
		if height := m.helpHeight(); height != 1 {
			t.Fatalf("height = %d, want 1", height)
		}

		m.Update(tcell.NewEventKey(tcell.KeyRune, ".", tcell.ModCtrl))
		if !m.help.ShowAll() {
			t.Fatal("toggle_help did not enable full help")
		}
	})

	t.Run("disabled", func(t *testing.T) {
		m := NewModel(cfg)

		if m.help != nil {
			t.Fatal("help model was created")
		}
		if count := m.rootFlex.GetItemCount(); count != 0 {
			t.Fatalf("layout item count = %d, want 0", count)
		}

		m.Update(tcell.NewEventKey(tcell.KeyRune, ".", tcell.ModCtrl))
		if m.help != nil {
			t.Fatal("toggle_help created help model")
		}
	})
}
