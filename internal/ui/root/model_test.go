package root

import (
	"path/filepath"
	"testing"

	"github.com/ayn2op/discordo/internal/config"
	"github.com/gdamore/tcell/v3"
)

func TestNewModel(t *testing.T) {
	cfg, err := config.Load(filepath.Join(t.TempDir(), "missing.toml"))
	if err != nil {
		t.Fatal(err)
	}

	t.Run("help enabled", func(t *testing.T) {
		cfg := *cfg
		cfg.Help.Enabled = true
		m := NewModel(&cfg)
		if !m.helpVisible {
			t.Fatal("help is hidden")
		}
		if height := m.helpView().Rows(0); height != 1 {
			t.Fatalf("height = %d, want 1", height)
		}

		m.Update(tcell.NewEventKey(tcell.KeyRune, ".", tcell.ModCtrl))
		if !m.helpShowAll {
			t.Fatal("toggle_full_help did not enable full help")
		}

		m.Update(tcell.NewEventKey(tcell.KeyRune, ".", tcell.ModAlt))
		if m.helpVisible {
			t.Fatal("help is visible")
		}

		m.Update(tcell.NewEventKey(tcell.KeyRune, ".", tcell.ModAlt))
		if !m.helpVisible {
			t.Fatal("help is hidden")
		}
	})

	t.Run("help disabled", func(t *testing.T) {
		cfg := *cfg
		cfg.Help.Enabled = false
		m := NewModel(&cfg)
		if m.helpVisible {
			t.Fatal("help is visible")
		}

		m.Update(tcell.NewEventKey(tcell.KeyRune, ".", tcell.ModCtrl))
		if m.helpShowAll {
			t.Fatal("toggle_full_help enabled full help")
		}

		m.Update(tcell.NewEventKey(tcell.KeyRune, ".", tcell.ModAlt))
		if m.helpVisible {
			t.Fatal("help is visible")
		}
	})
}
