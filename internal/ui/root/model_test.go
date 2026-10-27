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
		t.Run("shown", func(t *testing.T) {
			if !m.helpVisible {
				t.Fatal("help is hidden")
			}
		})
		t.Run("toggle full help", func(t *testing.T) {
			if m, _ = m.Update(tcell.NewEventKey(tcell.KeyRune, ".", tcell.ModCtrl)); !m.helpShowAll {
				t.Fatal("toggle_full_help did not enable full help")
			}
		})
		t.Run("toggle help hides it", func(t *testing.T) {
			if m, _ = m.Update(tcell.NewEventKey(tcell.KeyRune, ".", tcell.ModAlt)); m.helpVisible {
				t.Fatal("help is visible")
			}
		})
		t.Run("toggle help shows it again", func(t *testing.T) {
			if m, _ = m.Update(tcell.NewEventKey(tcell.KeyRune, ".", tcell.ModAlt)); !m.helpVisible {
				t.Fatal("help is hidden")
			}
		})
	})

	t.Run("help disabled", func(t *testing.T) {
		cfg := *cfg
		cfg.Help.Enabled = false
		m := NewModel(&cfg)
		t.Run("hidden", func(t *testing.T) {
			if m.helpVisible {
				t.Fatal("help is visible")
			}
		})
		t.Run("toggle full help does nothing", func(t *testing.T) {
			if m, _ = m.Update(tcell.NewEventKey(tcell.KeyRune, ".", tcell.ModCtrl)); m.helpShowAll {
				t.Fatal("toggle_full_help enabled full help")
			}
		})
		t.Run("toggle help does nothing", func(t *testing.T) {
			if m, _ = m.Update(tcell.NewEventKey(tcell.KeyRune, ".", tcell.ModAlt)); m.helpVisible {
				t.Fatal("help is visible")
			}
		})
	})
}

func TestModelQuit(t *testing.T) {
	cfg, err := config.Load(filepath.Join(t.TempDir(), "missing.toml"))
	if err != nil {
		t.Fatal(err)
	}
	m := NewModel(cfg)

	m, cmd := m.Update(tcell.NewEventKey(tcell.KeyCtrlC, "", tcell.ModCtrl))
	t.Run("asks", func(t *testing.T) {
		if cmd == nil {
			t.Fatal("quit did not ask")
		}
	})
	if cmd != nil {
		m, _ = m.Update(cmd())
	}
	t.Run("confirmation", func(t *testing.T) {
		if m.dialog == nil || m.dialog.Text != "Are you sure you want to quit?" {
			t.Fatalf("dialog = %+v, want the quit confirmation", m.dialog)
		}
	})
	t.Run("no", func(t *testing.T) {
		if no, _ := m.Update(dialogDoneMsg(0)); no.dialog != nil {
			t.Fatal("No left the dialog open")
		}
	})
	t.Run("yes", func(t *testing.T) {
		if _, cmd := m.Update(dialogDoneMsg(1)); cmd == nil {
			t.Fatal("Yes did not quit")
		}
	})
}
