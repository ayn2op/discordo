package form

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

	m := New("Login", Field{Label: "Login"}, Field{Label: "Password", Mask: "*"})
	send(&m, key(tcell.KeyRune, "a"))
	send(&m, key(tcell.KeyEnter, ""))
	send(&m, key(tcell.KeyRune, "b"))
	t.Run("typing fills the focused field and Enter moves on", func(t *testing.T) {
		if m.Value(0) != "a" || m.Value(1) != "b" {
			t.Fatalf("values = %q, %q", m.Value(0), m.Value(1))
		}
	})
	t.Run("enter on the button submits", func(t *testing.T) {
		send(&m, key(tcell.KeyTab, ""))
		if got := send(&m, key(tcell.KeyEnter, "")); got != (SubmitMsg{}) {
			t.Fatalf("got %v", got)
		}
	})
}
