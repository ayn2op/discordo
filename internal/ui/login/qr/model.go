package qr

import (
	"crypto/rsa"
	"strings"
	"time"

	"github.com/ayn2op/tview"
	"github.com/ayn2op/tview/tabs"
	"github.com/ayn2op/tview/text"
	"github.com/gdamore/tcell/v3"
	"github.com/gorilla/websocket"
	"github.com/skip2/go-qrcode"
)

type Model struct {
	*tview.TextView

	conn              *websocket.Conn
	heartbeatInterval time.Duration
	privateKey        *rsa.PrivateKey
	fingerprint       string

	qrCode *qrcode.QRCode
	status string
}

func NewModel() *Model {
	m := &Model{
		TextView: tview.NewTextView(),
	}
	m.
		SetScrollable(true).
		SetWrap(false).
		SetTextAlign(tview.AlignmentCenter)

	m.setStatus("Press Ctrl+N to open QR login")
	return m
}

var _ tabs.Tab = (*Model)(nil)

func (m *Model) Label() string {
	return "QR"
}

func (m *Model) Update(msg tview.Msg) tview.Cmd {
	switch msg := msg.(type) {
	case tview.InitMsg:
		m.setStatus("Connecting to Remote Auth Gateway...")
		return connect()
	case tview.KeyMsg:
		if msg.Key() == tcell.KeyEsc {
			m.setStatus("Canceled")
			return closeConn(m.conn)
		}
		return m.TextView.Update(msg)

	case connCreateMsg:
		m.conn = msg.conn
		m.setStatus("Connected. Handshaking...")
		return listen(m.conn)
	case connCloseMsg:
		m.conn = nil
		return nil

	case helloMsg:
		m.heartbeatInterval = time.Duration(msg.heartbeatInterval) * time.Millisecond
		return tview.Batch(listen(m.conn), heartbeat(m.heartbeatInterval), generatePrivateKey())
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
		return tview.Batch(heartbeat(m.heartbeatInterval), sendHeartbeat(m.conn))

	case errMsg:
		m.setStatus(msg.Error())
		return closeConn(m.conn)
	}

	return nil
}

// halfBlock packs a vertical pair of QR pixels into one glyph, fitting two
// bitmap rows into a single terminal row.
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

func (m *Model) SetRect(x, y, width, height int) {
	_, _, _, oldHeight := m.Rect()
	m.TextView.SetRect(x, y, width, height)
	if oldHeight != height {
		m.render()
	}
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

	builder := new(text.Builder)
	builder.Write(out.String(), tcell.StyleDefault)
	m.SetContent(m.centerText(builder.Finish()))
	m.TextView.SetRect(m.Rect())
}

func (m *Model) View(screen tcell.Screen) {
	m.TextView.View(screen)
}

func (m *Model) centerText(content text.Text) text.Text {
	_, _, _, height := m.InnerRect()
	if height == 0 {
		height = 40
	}
	padding := (height - len(content)) / 2
	if padding < 0 {
		padding = 0
	} else if padding < 1 && height > len(content) {
		padding = 1
	}
	if padding == 0 {
		return content
	}

	centered := make(text.Text, 0, padding+len(content))
	centered = append(centered, make(text.Text, padding)...)
	centered = append(centered, content...)
	return centered
}
