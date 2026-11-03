package root

import (
	"testing"

	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/tview"
	"github.com/ayn2op/tview/cmd"
	"github.com/gdamore/tcell/v3"
	"github.com/gdamore/tcell/v3/vt"
)

func TestNewModel(t *testing.T) {
	cfg, err := config.Load(t.TempDir())
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
	cfg, err := config.Load(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	m := NewModel(cfg)

	m, ask := m.Update(tcell.NewEventKey(tcell.KeyCtrlC, "", tcell.ModCtrl))
	t.Run("asks", func(t *testing.T) {
		if ask == nil {
			t.Fatal("quit did not ask")
		}
	})
	if ask != nil {
		m, _ = m.Update(ask.(cmd.Task)())
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

func TestModelView(t *testing.T) {
	cfg, err := config.Load(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	cfg.Theme = "tokyonight"

	screen, err := tcell.NewTerminfoScreenFromTty(vt.NewMockTerm(vt.MockOptSize{X: 40, Y: 10}))
	if err != nil {
		t.Fatal(err)
	}
	if err := screen.Init(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(screen.Fini)

	NewModel(cfg).View().Draw(screen, tview.Rectangle{Width: 40, Height: 10})
	want := cfg.Palette().Background.Base.GetBackground()
	if _, style, _ := screen.Get(39, 9); style.GetBackground() != want {
		t.Fatalf("background = %v, want the theme's", style.GetBackground())
	}
}
