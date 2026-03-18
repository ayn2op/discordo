package qr

import (
	"crypto/rsa"
	"fmt"
	"strings"
	"time"

	"github.com/ayn2op/tview"
	"github.com/ayn2op/tview/tabs"
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
	msg    string
}

func NewModel() *Model {
	m := &Model{
		TextView: tview.NewTextView(),
	}
	m.
		SetScrollable(true).
		SetWrap(false).
		SetTextAlign(tview.AlignmentCenter)

	m.msg = "Press Ctrl+N to open QR login"
	return m
}

var _ tabs.Tab = (*Model)(nil)

func (m *Model) Label() string {
	return "QR"
}

func (m *Model) HandleEvent(event tview.Event) tview.Command {
	switch event := event.(type) {
	case *tview.InitEvent:
		m.msg = "Connecting to Remote Auth Gateway..."
		return m.connect()
	case *tview.KeyEvent:
		if event.Key() == tcell.KeyEsc {
			m.msg = "Canceled"
			return tview.Batch(m.close(), nil)
		}
		return m.TextView.HandleEvent(event)

	case *connCreateEvent:
		m.conn = event.conn
		m.msg = "Connected. Handshaking..."
		return m.listen()
	case *connCloseEvent:
		m.conn = nil
		return nil

	case *helloEvent:
		m.heartbeatInterval = time.Duration(event.heartbeatInterval) * time.Millisecond
		return tview.Batch(m.listen(), m.heartbeat(), m.generatePrivateKey())
	case *privateKeyEvent:
		m.privateKey = event.privateKey
		return tview.Batch(m.listen(), m.sendInit())
	case *nonceProofEvent:
		return tview.Batch(m.listen(), m.sendNonceProof(event.encryptedNonce))
	case *pendingRemoteInitEvent:
		m.fingerprint = event.fingerprint
		return tview.Batch(m.listen(), m.generateQRCode(event.fingerprint))
	case *qrCodeEvent:
		m.qrCode = event.qrCode
		m.msg = "Scan this with the Discord mobile app to log in instantly."
		return m.listen()
	case *pendingTicketEvent:
		return tview.Batch(m.listen(), m.decryptUserPayload(event.encryptedUserPayload))
	case *userEvent:
		name := event.username
		if event.discriminator != "0" {
			name += "#" + event.discriminator
		}
		m.msg = fmt.Sprintf("Check your phone! Logging in as %s", name)
		return m.listen()
	case *pendingLoginEvent:
		m.msg = "Authenticating..."
		return tview.Batch(m.close(), m.exchangeTicket(event.ticket))
	case *cancelEvent:
		m.msg = "Login canceled on mobile"
		return m.close()

	case *heartbeatTickEvent:
		if m.conn == nil {
			return nil
		}
		return tview.Batch(m.heartbeat(), m.sendHeartbeat())

	case *tcell.EventError:
		m.msg = event.Error()
		return tview.Batch(m.close(), tview.Command(func() tview.Event { return event }))
	}

	return nil
}

func (m *Model) Draw(screen tcell.Screen) {
	var contents []string
	if m.qrCode != nil {
		bitmap := m.qrCode.Bitmap()
		var b strings.Builder
		for y := 0; y < len(bitmap); y += 2 {
			for x := range bitmap[y] {
				top := bitmap[y][x]
				bottom := false
				if y+1 < len(bitmap) {
					bottom = bitmap[y+1][x]
				}
				if top && bottom {
					b.WriteString("█")
				} else if top && !bottom {
					b.WriteString("▀")
				} else if !top && bottom {
					b.WriteString("▄")
				} else {
					b.WriteByte(' ')
				}
			}
			b.WriteByte('\n')
		}
		contents = append(contents, b.String())
	}
	if m.msg != "" {
		contents = append(contents, m.msg)
	}

	builder := tview.NewLineBuilder()
	builder.Write(strings.Join(contents, "\n"), tcell.StyleDefault)
	m.SetLines(m.centerLines(builder.Finish()))
	m.TextView.Draw(screen)
}

func (m *Model) centerLines(lines []tview.Line) []tview.Line {
	_, _, _, height := m.InnerRect()
	if height == 0 {
		height = 40
	}
	padding := (height - len(lines)) / 2
	if padding < 0 {
		padding = 0
	} else if padding < 1 && height > len(lines) {
		padding = 1
	}
	if padding == 0 {
		return lines
	}

	centered := make([]tview.Line, 0, padding+len(lines))
	centered = append(centered, make([]tview.Line, padding)...)
	centered = append(centered, lines...)
	return centered
}
