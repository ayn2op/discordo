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
		m, _ = m.Update(tcell.NewEventKey(tcell.KeyRune, ".", tcell.ModCtrl))
		if !m.helpShowAll {
			t.Fatal("toggle_full_help did not enable full help")
		}

		m, _ = m.Update(tcell.NewEventKey(tcell.KeyRune, ".", tcell.ModAlt))
		if m.helpVisible {
			t.Fatal("help is visible")
		}

		m, _ = m.Update(tcell.NewEventKey(tcell.KeyRune, ".", tcell.ModAlt))
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

		m, _ = m.Update(tcell.NewEventKey(tcell.KeyRune, ".", tcell.ModCtrl))
		if m.helpShowAll {
			t.Fatal("toggle_full_help enabled full help")
		}

		m, _ = m.Update(tcell.NewEventKey(tcell.KeyRune, ".", tcell.ModAlt))
		if m.helpVisible {
			t.Fatal("help is visible")
		}
	})
}

func TestModelQuit(t *testing.T) {
	cfg, err := config.Load(filepath.Join(t.TempDir(), "missing.toml"))
	if err != nil {
		t.Fatal(err)
	}
	m := NewModel(cfg)

	m, cmd := m.Update(tcell.NewEventKey(tcell.KeyCtrlC, "", tcell.ModCtrl))
	if cmd == nil {
		t.Fatal("quit did not ask")
	}
	m, _ = m.Update(cmd())
	if m.dialog == nil || m.dialog.Text != "Are you sure you want to quit?" {
		t.Fatalf("dialog = %+v, want the quit confirmation", m.dialog)
	}
	if no, _ := m.Update(dialogDoneMsg(0)); no.dialog != nil {
		t.Fatal("No left the dialog open")
	}
	if _, cmd = m.Update(dialogDoneMsg(1)); cmd == nil {
		t.Fatal("Yes did not quit")
	}
}
