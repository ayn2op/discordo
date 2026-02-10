package qr

import (
	"crypto/rsa"
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/ayn2op/discordo/pkg/tabs"
	"github.com/gorilla/websocket"
	"github.com/skip2/go-qrcode"
)

type Model struct {
	conn              *websocket.Conn
	heartbeatInterval time.Duration
	privateKey        *rsa.PrivateKey
	fingerprint       string

	qrCode *qrcode.QRCode
	msg    string
}

func NewModel() Model {
	return Model{}
}

var _ tabs.Tab = Model{}

func (m Model) Label() string {
	return "QR"
}

var _ tea.Model = Model{}

func (m Model) Init() tea.Cmd {
	return m.connect()
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case connCreateMsg:
		m.conn = msg
		m.msg = "Successfully connected to the remote authentication gateway."
		return m, m.listen()
	case connCloseMsg:
		m.conn = nil
		return m, nil

	case helloMsg:
		m.heartbeatInterval = time.Duration(msg.HeartbeatInterval) * time.Millisecond
		return m, tea.Batch(m.listen(), m.heartbeat(), m.generatePrivateKey())
	case privateKeyMsg:
		m.privateKey = msg
		return m, tea.Batch(m.listen(), m.sendInit())
	case nonceProofMsg:
		return m, tea.Batch(m.listen(), m.sendNonceProof(msg.EncryptedNonce))
	case pendingRemoteInitMsg:
		m.fingerprint = msg.Fingerprint
		return m, tea.Batch(m.listen(), m.generateQRCode(msg.Fingerprint))
	case qrCodeMsg:
		m.qrCode = msg
		m.msg = "Scan this with the Discord mobile app to log in instantly."
		return m, m.listen()
	case pendingTicketMsg:
		return m, tea.Batch(m.listen(), m.decryptUserPayload(msg.EncryptedUserPayload))
	case userMsg:
		name := msg.Username
		if msg.Discriminator != "0" {
			name += "#" + msg.Discriminator
		}
		m.msg = fmt.Sprintf("Check your phone! Logging in as %s", name)
		return m, m.listen()
	case pendingLoginMsg:
		return m, tea.Batch(m.close(), m.exchangeTicket(msg.Ticket))
	case cancelMsg:
		return m, m.close()

	case heartbeatTickMsg:
		if m.conn == nil {
			return m, nil // Stop the heartbeat chain
		}
		return m, tea.Batch(m.heartbeat(), m.sendHeartbeat())

	case error:
		m.msg = msg.Error()
		return m, m.close()
	}
	return m, nil
}

func (m Model) View() tea.View {
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
	return tea.NewView(lipgloss.JoinVertical(lipgloss.Center, contents...))
}
