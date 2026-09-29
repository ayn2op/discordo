package qr

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ayn2op/tview"
	"github.com/gdamore/tcell/v3"
	"github.com/gdamore/tcell/v3/vt"
	"github.com/gorilla/websocket"
)

// gatewayModel runs Model on an open connection and quits once the login is canceled.
type gatewayModel struct {
	Model
	conn *websocket.Conn
}

func (m gatewayModel) Init() tview.Cmd {
	return func() tview.Msg { return connCreateMsg{conn: &gatewayConn{ws: m.conn}} }
}

func (m gatewayModel) Update(msg tview.Msg) (gatewayModel, tview.Cmd) {
	var cmd tview.Cmd
	m.Model, cmd = m.Model.Update(msg)
	if m.status == "Login canceled on mobile" {
		return m, tview.Quit()
	}
	return m, cmd
}

// serveGateway fakes the Remote Auth Gateway, heartbeating every millisecond to overlap the client's writes, and cancels after acks heartbeat acks.
func serveGateway(t *testing.T, acks int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := (&websocket.Upgrader{}).Upgrade(w, r, nil)
		if err != nil {
			t.Error(err)
			return
		}
		defer conn.Close()

		if err := conn.WriteJSON(map[string]any{"op": "hello", "heartbeat_interval": 1}); err != nil {
			return
		}
		shown := false
		for {
			var msg struct {
				Op string `json:"op"`
			}
			err := conn.ReadJSON(&msg)
			switch {
			case err != nil:
			case msg.Op == "heartbeat":
				if shown {
					acks--
				}
				err = conn.WriteJSON(map[string]any{"op": "heartbeat_ack"})
			case msg.Op == "init":
				shown = true
				err = conn.WriteJSON(map[string]any{"op": "pending_remote_init", "fingerprint": "fingerprint"})
			}
			if err != nil {
				return
			}
			if shown && acks <= 0 {
				_ = conn.WriteJSON(map[string]any{"op": "cancel"})
				return
			}
		}
	}
}

func TestModelGateway(t *testing.T) {
	t.Run("reads messages after heartbeat acks", func(t *testing.T) {
		server := httptest.NewServer(serveGateway(t, 5))
		t.Cleanup(server.Close)
		conn, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(server.URL, "http"), nil)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { conn.Close() })

		screen, err := tcell.NewTerminfoScreenFromTty(vt.NewMockTerm(vt.MockOptSize{X: 80, Y: 24}))
		if err != nil {
			t.Fatal(err)
		}
		if err := screen.Init(); err != nil {
			t.Fatal(err)
		}

		done := make(chan error, 1)
		go func() {
			done <- tview.NewApplication(gatewayModel{Model: NewModel(), conn: conn}, tview.WithScreen(screen)).Run()
		}()
		select {
		case err := <-done:
			if err != nil {
				t.Fatal(err)
			}
		case <-time.After(5 * time.Second):
			t.Fatal("the cancel from the gateway was never read")
		}
	})
}
