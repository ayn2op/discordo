package password

import (
	"testing"

	"github.com/ayn2op/tview"
	"github.com/gdamore/tcell/v3"
)

func TestModelUpdate(t *testing.T) {
	area := tview.Rectangle{Width: 30, Height: 5}
	// send delivers msg the way the application does: through the view, then Update.
	send := func(m *Model, msg tview.Msg) tview.Msg {
		if msg = m.View().Handle(msg, area); msg != nil {
			*m, _ = m.Update(msg)
		}
		return msg
	}
	key := func(k tcell.Key, str string) tview.KeyMsg { return tcell.NewEventKey(k, str, tcell.ModNone) }

	var m Model
	send(&m, key(tcell.KeyRune, "a"))
	send(&m, key(tcell.KeyEnter, ""))
	send(&m, key(tcell.KeyRune, "b"))
	t.Run("typing fills the focused input and Enter moves on", func(t *testing.T) {
		if m.login != "a" || m.password != "b" {
			t.Fatalf("values = %q, %q", m.login, m.password)
		}
	})
	t.Run("enter on the button submits", func(t *testing.T) {
		send(&m, key(tcell.KeyTab, ""))
		if got := send(&m, key(tcell.KeyEnter, "")); got != (submitMsg{}) {
			t.Fatalf("got %v", got)
		}
	})

	t.Run("a second factor switches to the code stage", func(t *testing.T) {
		var m Model
		m, _ = m.Update(mfaMsg{mfaMethodTOTP, "ticket", "instance"})
		if m.stage != stageCode || m.ticket != "ticket" || m.loginInstanceID != "instance" {
			t.Fatalf("stage = %v, ticket = %q, loginInstanceID = %q", m.stage, m.ticket, m.loginInstanceID)
		}
	})

	t.Run("submitting an empty code does nothing", func(t *testing.T) {
		m := Model{stage: stageCode}
		if _, cmd := m.Update(submitMsg{}); cmd != nil {
			t.Fatal("expected no command for an empty code")
		}
	})

	t.Run("submitting a code submits", func(t *testing.T) {
		m := Model{stage: stageCode, code: "123456"}
		if _, cmd := m.Update(submitMsg{}); cmd == nil {
			t.Fatal("expected a command for a non-empty code")
		}
	})

	t.Run("tab cycles between the code input and the button only in the code stage", func(t *testing.T) {
		m := Model{stage: stageCode}
		m, _ = m.Update(tcell.NewEventKey(tcell.KeyTab, "", tcell.ModNone))
		if m.focus != focusCodeButton {
			t.Fatalf("focus = %v, want focusCodeButton", m.focus)
		}
		m, _ = m.Update(tcell.NewEventKey(tcell.KeyTab, "", tcell.ModNone))
		if m.focus != focusCode {
			t.Fatalf("focus = %v, want focusCode", m.focus)
		}
	})
}
