package qr

import (
	"crypto/rsa"
	"github.com/ayn2op/tview/layout"
	"strings"
	"time"

	"github.com/ayn2op/tview"
	"github.com/ayn2op/tview/center"
	"github.com/ayn2op/tview/richtext"
	"github.com/ayn2op/tview/textview"
	"github.com/gdamore/tcell/v3"
	"github.com/skip2/go-qrcode"
)

type Model struct {
	scrollState textview.ScrollState

	conn              *gatewayConn
	heartbeatInterval time.Duration
	privateKey        *rsa.PrivateKey
	fingerprint       string

	qrCode *qrcode.QRCode
	status string
}

func NewModel() Model {
	var m Model
	m.status = "Press Ctrl+N to open QR login"
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
		m.status = "Connecting to Remote Auth Gateway..."
		return connect()
	case tview.KeyMsg:
		if msg.Key() == tcell.KeyEsc {
			m.status = "Canceled"
			return closeConn(m.conn)
		}
		return nil
	case scrollMsg:
		m.scrollState.Apply(textview.Change(msg))
		return nil

	case connCreateMsg:
		m.conn = msg.conn
		m.status = "Connected. Handshaking..."
		return listen(m.conn)
	case connCloseMsg:
		m.conn = nil
		return nil

	case helloMsg:
		m.heartbeatInterval = time.Duration(msg.heartbeatInterval) * time.Millisecond
		return tview.Batch(listen(m.conn), scheduleHeartbeat(m.heartbeatInterval), generatePrivateKey())
	case privateKeyMsg:
		m.privateKey = msg.privateKey
		return sendInit(m.conn, m.privateKey)
	case nonceProofMsg:
		return tview.Batch(listen(m.conn), sendNonceProof(m.conn, m.privateKey, msg.encryptedNonce))
	case pendingRemoteInitMsg:
		m.fingerprint = msg.fingerprint
		return tview.Batch(listen(m.conn), generateQRCode(msg.fingerprint))
	case qrCodeMsg:
		m.qrCode = msg.qrCode
		m.status = "Scan this with the Discord mobile app to log in instantly."
		return nil
	case pendingTicketMsg:
		return tview.Batch(listen(m.conn), decryptUserPayload(m.privateKey, msg.encryptedUserPayload))
	case userMsg:
		name := msg.username
		if msg.discriminator != "0" {
			name += "#" + msg.discriminator
		}
		m.status = "Check your phone! Logging in as " + name
		return nil
	case pendingLoginMsg:
		m.status = "Authenticating..."
		return tview.Batch(closeConn(m.conn), exchangeTicket(m.fingerprint, m.privateKey, msg.ticket))
	case ignoredMsg:
		return listen(m.conn)
	case cancelMsg:
		m.status = "Login canceled on mobile"
		return closeConn(m.conn)

	case heartbeatTickMsg:
		if m.conn == nil {
			return nil
		}
		return tview.Batch(scheduleHeartbeat(m.heartbeatInterval), sendHeartbeat(m.conn))

	case error:
		m.status = msg.Error()
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
func (m Model) View() tview.Widget {
	code := m.code()
	return center.New(
		textview.New(code).
			ScrollState(&m.scrollState).
			Wrap(false).
			Alignment(tview.AlignmentCenter).
			Height(layout.Fixed(len(code))).
			Focused(true).
			OnChange(func(a textview.Change) tview.Msg { return scrollMsg(a) }),
	)
}

// code returns the QR code above the status.
func (m Model) code() richtext.Text {
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
	return builder.Finish()
}
