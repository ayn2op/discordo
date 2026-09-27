package qr

import (
	"crypto/rsa"
	"strings"
	"time"

	"github.com/ayn2op/tview"
	"github.com/ayn2op/tview/center"
	"github.com/ayn2op/tview/richtext"
	"github.com/ayn2op/tview/textview"
	"github.com/gdamore/tcell/v3"
	"github.com/gorilla/websocket"
	"github.com/skip2/go-qrcode"
)

type Model struct {
	// code is the QR code above the status, scrolled when it does not fit.
	code        richtext.Text
	scrollState textview.ScrollState

	conn              *websocket.Conn
	heartbeatInterval time.Duration
	privateKey        *rsa.PrivateKey
	fingerprint       string

	qrCode *qrcode.QRCode
	status string
}

// scrollMsg scrolls the code.
type scrollMsg textview.Action

// startMsg starts connecting to the Remote Auth Gateway.
type startMsg struct{}

func NewModel() Model {
	var m Model
	m.setStatus("Press Ctrl+N to open QR login")
	return m
}

var _ tview.Model[Model] = Model{}

func (Model) Label() string {
	return "QR"
}

func (Model) Init() tview.Cmd {
	return func() tview.Msg { return startMsg{} }
}

func (m Model) Update(msg tview.Msg) (Model, tview.Cmd) {
	cmd := m.update(msg)
	return m, cmd
}

// update changes m in response to msg and returns a command to run, or nil.
func (m *Model) update(msg tview.Msg) tview.Cmd {
	switch msg := msg.(type) {
	case startMsg:
		m.setStatus("Connecting to Remote Auth Gateway...")
		return connect()
	case tview.KeyMsg:
		if msg.Key() == tcell.KeyEsc {
			m.setStatus("Canceled")
			return closeConn(m.conn)
		}
		return nil
	case scrollMsg:
		m.scrollState.Perform(textview.Action(msg))
		return nil

	case connCreateMsg:
		m.conn = msg.conn
		m.setStatus("Connected. Handshaking...")
		return listen(m.conn)
	case connCloseMsg:
		m.conn = nil
		return nil

	case helloMsg:
		m.heartbeatInterval = time.Duration(msg.heartbeatInterval) * time.Millisecond
		return tview.Batch(listen(m.conn), scheduleHeartbeat(m.heartbeatInterval), generatePrivateKey())
	case privateKeyMsg:
		m.privateKey = msg.privateKey
		return tview.Batch(listen(m.conn), sendInit(m.conn, m.privateKey))
	case nonceProofMsg:
		return tview.Batch(listen(m.conn), sendNonceProof(m.conn, m.privateKey, msg.encryptedNonce))
	case pendingRemoteInitMsg:
		m.fingerprint = msg.fingerprint
		return tview.Batch(listen(m.conn), generateQRCode(msg.fingerprint))
	case qrCodeMsg:
		m.qrCode = msg.qrCode
		m.setStatus("Scan this with the Discord mobile app to log in instantly.")
		return listen(m.conn)
	case pendingTicketMsg:
		return tview.Batch(listen(m.conn), decryptUserPayload(m.privateKey, msg.encryptedUserPayload))
	case userMsg:
		name := msg.username
		if msg.discriminator != "0" {
			name += "#" + msg.discriminator
		}
		m.setStatus("Check your phone! Logging in as " + name)
		return listen(m.conn)
	case pendingLoginMsg:
		m.setStatus("Authenticating...")
		return tview.Batch(closeConn(m.conn), exchangeTicket(m.fingerprint, m.privateKey, msg.ticket))
	case cancelMsg:
		m.setStatus("Login canceled on mobile")
		return closeConn(m.conn)

	case heartbeatTickMsg:
		if m.conn == nil {
			return nil
		}
		return tview.Batch(scheduleHeartbeat(m.heartbeatInterval), sendHeartbeat(m.conn))

	case errMsg:
		m.setStatus(msg.Error())
		return closeConn(m.conn)
	}

	return nil
}

// halfBlock packs a vertical pair of QR pixels into one glyph, fitting two bitmap rows into a single terminal row.
func halfBlock(top, bottom bool) rune {
	switch [2]bool{top, bottom} {
	case [2]bool{true, true}:
		return '█'
	case [2]bool{true, false}:
		return '▀'
	case [2]bool{false, true}:
		return '▄'
	default:
		return ' '
	}
}

// View centers the code, which scrolls when it is taller than the tab.
func (m Model) View() tview.Element {
	return center.New(
		textview.New(m.code).
			ScrollState(&m.scrollState).
			Wrap(false).
			Alignment(tview.AlignmentCenter).
			Height(tview.Fixed(len(m.code))).
			Focused(true).
			OnAction(func(a textview.Action) tview.Msg { return scrollMsg(a) }),
	)
}

func (m *Model) setStatus(status string) {
	m.status = status
	m.render()
}

func (m *Model) render() {
	var out strings.Builder
	if m.qrCode != nil {
		bitmap := m.qrCode.Bitmap()
		for y := 0; y < len(bitmap); y += 2 {
			for x := range bitmap[y] {
				top := bitmap[y][x]
				bottom := y+1 < len(bitmap) && bitmap[y+1][x]
				out.WriteRune(halfBlock(top, bottom))
			}
			out.WriteByte('\n')
		}
	}
	if m.status != "" {
		if out.Len() > 0 {
			out.WriteByte('\n')
		}
		out.WriteString(m.status)
	}

	builder := new(richtext.Builder)
	builder.Write(out.String(), tcell.StyleDefault)
	m.code = builder.Finish()
}
