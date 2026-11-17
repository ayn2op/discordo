package reminderprompt

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/tview"
)

// newTestModel returns a prompt opened on onSubmit, with the default config.
func newTestModel(t *testing.T, onSubmit func(text string) (tview.Cmd, error)) *Model {
	t.Helper()
	cfg, err := config.Load(filepath.Join(t.TempDir(), "config.toml"))
	if err != nil {
		t.Fatal(err)
	}
	m := NewModel(cfg)
	m.Open(onSubmit)
	return &m
}

// submitted reports what SubmittedMsg cmd carries, or nil if it is not one.
func submitted(t *testing.T, cmd tview.Cmd) *SubmittedMsg {
	t.Helper()
	if cmd == nil {
		return nil
	}
	msg, ok := cmd().(SubmittedMsg)
	if !ok {
		return nil
	}
	return &msg
}

func TestModelSubmit(t *testing.T) {
	accept := func(string) (tview.Cmd, error) { return func() tview.Msg { return nil }, nil }
	reject := func(string) (tview.Cmd, error) { return nil, errors.New("unreadable") }

	t.Run("accepted", func(t *testing.T) {
		m := newTestModel(t, accept)
		*m, _ = m.Update(inputMsg{"45m", len("45m")})
		var cmd tview.Cmd
		*m, cmd = m.Update(submitMsg{})
		if submitted(t, cmd) == nil {
			t.Fatal("an accepted time did not submit")
		}
		if m.footer() != hint {
			t.Fatalf("footer = %q, want the hint", m.footer())
		}
	})

	t.Run("rejected keeps the text", func(t *testing.T) {
		m := newTestModel(t, reject)
		*m, _ = m.Update(inputMsg{"next tuesday", len("next tuesday")})
		var cmd tview.Cmd
		*m, cmd = m.Update(submitMsg{})
		if cmd != nil {
			t.Fatalf("a rejected time submitted: %v", cmd())
		}
		if m.text != "next tuesday" {
			t.Fatalf("text = %q, want it kept for editing", m.text)
		}
		if want := "unreadable: " + hint; m.footer() != want {
			t.Fatalf("footer = %q, want %q", m.footer(), want)
		}
	})

	t.Run("editing drops the rejection", func(t *testing.T) {
		m := newTestModel(t, reject)
		*m, _ = m.Update(inputMsg{"x", 1})
		*m, _ = m.Update(submitMsg{})
		*m, _ = m.Update(inputMsg{"xy", 2})
		if m.footer() != hint {
			t.Fatalf("footer = %q, want the hint back", m.footer())
		}
	})

	t.Run("empty", func(t *testing.T) {
		m := newTestModel(t, func(string) (tview.Cmd, error) {
			t.Fatal("an empty input was submitted")
			return nil, nil
		})
		*m, _ = m.Update(inputMsg{"  ", 2})
		var cmd tview.Cmd
		*m, cmd = m.Update(submitMsg{})
		if cmd != nil {
			t.Fatalf("an empty input produced a command: %v", cmd())
		}
	})
}

func TestModelOpen(t *testing.T) {
	m := newTestModel(t, func(string) (tview.Cmd, error) { return nil, errors.New("unreadable") })
	*m, _ = m.Update(inputMsg{"45m", len("45m")})
	*m, _ = m.Update(submitMsg{})

	m.Open(func(string) (tview.Cmd, error) { return nil, nil })
	if m.text != "" || m.cursor != 0 || m.footer() != hint {
		t.Fatalf("reopened with text %q, cursor %d and footer %q", m.text, m.cursor, m.footer())
	}
}
