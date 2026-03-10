package qr

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/ayn2op/discordo/internal/http"
	"github.com/ayn2op/tview"
	"github.com/diamondburned/arikawa/v3/utils/httputil"
	"github.com/gdamore/tcell/v3"
	"github.com/gorilla/websocket"
	"github.com/skip2/go-qrcode"
)

type TokenEvent struct {
	tcell.EventTime
	Token string
}

func newTokenEvent(token string) *TokenEvent {
	event := &TokenEvent{Token: token}
	event.SetEventNow()
	return event
}

const remoteAuthGatewayURL = "wss://remote-auth-gateway.discord.gg/?v=2"

type connCreateEvent struct {
	tcell.EventTime
	conn *websocket.Conn
}

func newConnCreateEvent(conn *websocket.Conn) *connCreateEvent {
	event := &connCreateEvent{conn: conn}
	event.SetEventNow()
	return event
}

type connCloseEvent struct{ tcell.EventTime }

func newConnCloseEvent() *connCloseEvent {
	event := &connCloseEvent{}
	event.SetEventNow()
	return event
}

func (m *Model) connect() tview.Command {
	return tview.EventCommand(func() tcell.Event {
		headers := http.Headers()
		headers.Set("User-Agent", http.BrowserUserAgent)
		conn, _, err := websocket.DefaultDialer.Dial(remoteAuthGatewayURL, headers)
		if err != nil {
			return tcell.NewEventError(err)
		}
		return newConnCreateEvent(conn)
	})
}

func (m *Model) close() tview.Command {
	return tview.EventCommand(func() tcell.Event {
		if m.conn != nil {
			if err := m.conn.Close(); err != nil {
				return tcell.NewEventError(err)
			}
		}
		return newConnCloseEvent()
	})
}

type helloEvent struct {
	tcell.EventTime
	heartbeatInterval int
	timeoutMS         int
}

func newHelloEvent(heartbeatInterval, timeoutMS int) *helloEvent {
	event := &helloEvent{heartbeatInterval: heartbeatInterval, timeoutMS: timeoutMS}
	event.SetEventNow()
	return event
}

type nonceProofEvent struct {
	tcell.EventTime
	encryptedNonce string
}

func newNonceProofEvent(encryptedNonce string) *nonceProofEvent {
	event := &nonceProofEvent{encryptedNonce: encryptedNonce}
	event.SetEventNow()
	return event
}

type pendingRemoteInitEvent struct {
	tcell.EventTime
	fingerprint string
}

func newPendingRemoteInitEvent(fingerprint string) *pendingRemoteInitEvent {
	event := &pendingRemoteInitEvent{fingerprint: fingerprint}
	event.SetEventNow()
	return event
}

type pendingTicketEvent struct {
	tcell.EventTime
	encryptedUserPayload string
}

func newPendingTicketEvent(encryptedUserPayload string) *pendingTicketEvent {
	event := &pendingTicketEvent{encryptedUserPayload: encryptedUserPayload}
	event.SetEventNow()
	return event
}

type pendingLoginEvent struct {
	tcell.EventTime
	ticket string
}

func newPendingLoginEvent(ticket string) *pendingLoginEvent {
	event := &pendingLoginEvent{ticket: ticket}
	event.SetEventNow()
	return event
}

type cancelEvent struct{ tcell.EventTime }

func newCancelEvent() *cancelEvent {
	event := &cancelEvent{}
	event.SetEventNow()
	return event
}

func (m *Model) listen() tview.Command {
	return tview.EventCommand(func() tcell.Event {
		if m.conn == nil {
			return nil
		}

		_, data, err := m.conn.ReadMessage()
		if err != nil {
			return tcell.NewEventError(err)
		}

		var payload struct {
			Op string `json:"op"`
		}
		if err := json.Unmarshal(data, &payload); err != nil {
			return tcell.NewEventError(err)
		}

		switch payload.Op {
		case "hello":
			var payload struct {
				HeartbeatInterval int `json:"heartbeat_interval"`
				TimeoutMS         int `json:"timeout_ms"`
			}
			if err := json.Unmarshal(data, &payload); err != nil {
				return tcell.NewEventError(err)
			}
			return newHelloEvent(payload.HeartbeatInterval, payload.TimeoutMS)
		case "nonce_proof":
			var payload struct {
				EncryptedNonce string `json:"encrypted_nonce"`
			}
			if err := json.Unmarshal(data, &payload); err != nil {
				return tcell.NewEventError(err)
			}
			return newNonceProofEvent(payload.EncryptedNonce)
		case "pending_remote_init":
			var payload struct {
				Fingerprint string `json:"fingerprint"`
			}
			if err := json.Unmarshal(data, &payload); err != nil {
				return tcell.NewEventError(err)
			}
			return newPendingRemoteInitEvent(payload.Fingerprint)
		case "pending_ticket":
			var payload struct {
				EncryptedUserPayload string `json:"encrypted_user_payload"`
			}
			if err := json.Unmarshal(data, &payload); err != nil {
				return tcell.NewEventError(err)
			}
			return newPendingTicketEvent(payload.EncryptedUserPayload)
		case "cancel":
			return newCancelEvent()
		case "pending_login":
			var payload struct {
				Ticket string `json:"ticket"`
			}
			if err := json.Unmarshal(data, &payload); err != nil {
				return tcell.NewEventError(err)
			}
			return newPendingLoginEvent(payload.Ticket)
		default:
			return nil
		}
	})
}

type heartbeatTickEvent struct{ tcell.EventTime }

func newHeartbeatTickEvent() *heartbeatTickEvent {
	event := &heartbeatTickEvent{}
	event.SetEventNow()
	return event
}

func (m *Model) heartbeat() tview.Command {
	return tview.EventCommand(func() tcell.Event {
		time.Sleep(m.heartbeatInterval)
		return newHeartbeatTickEvent()
	})
}

func (m *Model) sendHeartbeat() tview.Command {
	return tview.EventCommand(func() tcell.Event {
		if m.conn == nil {
			return nil
		}
		data := struct {
			Op string `json:"op"`
		}{"heartbeat"}
		if err := m.conn.WriteJSON(data); err != nil {
			return tcell.NewEventError(err)
		}
		return nil
	})
}

type privateKeyEvent struct {
	tcell.EventTime
	privateKey *rsa.PrivateKey
}

func newPrivateKeyEvent(privateKey *rsa.PrivateKey) *privateKeyEvent {
	event := &privateKeyEvent{privateKey: privateKey}
	event.SetEventNow()
	return event
}

func (m *Model) generatePrivateKey() tview.Command {
	return tview.EventCommand(func() tcell.Event {
		privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			return tcell.NewEventError(err)
		}
		return newPrivateKeyEvent(privateKey)
	})
}

func (m *Model) sendInit() tview.Command {
	return tview.EventCommand(func() tcell.Event {
		if m.privateKey == nil {
			return tcell.NewEventError(errors.New("missing private key"))
		}
		spki, err := x509.MarshalPKIXPublicKey(m.privateKey.Public())
		if err != nil {
			return tcell.NewEventError(err)
		}
		encodedPublicKey := base64.StdEncoding.EncodeToString(spki)
		data := struct {
			Op               string `json:"op"`
			EncodedPublicKey string `json:"encoded_public_key"`
		}{"init", encodedPublicKey}
		if err := m.conn.WriteJSON(data); err != nil {
			return tcell.NewEventError(err)
		}
		return nil
	})
}

func (m *Model) sendNonceProof(encryptedNonce string) tview.Command {
	return tview.EventCommand(func() tcell.Event {
		decodedNonce, err := base64.StdEncoding.DecodeString(encryptedNonce)
		if err != nil {
			return tcell.NewEventError(err)
		}

		decryptedNonce, err := rsa.DecryptOAEP(sha256.New(), nil, m.privateKey, decodedNonce, nil)
		if err != nil {
			return tcell.NewEventError(err)
		}

		encodedNonce := base64.RawURLEncoding.EncodeToString(decryptedNonce)
		data := struct {
			Op    string `json:"op"`
			Nonce string `json:"nonce"`
		}{"nonce_proof", encodedNonce}
		if err := m.conn.WriteJSON(data); err != nil {
			return tcell.NewEventError(err)
		}
		return nil
	})
}

type qrCodeEvent struct {
	tcell.EventTime
	qrCode *qrcode.QRCode
}

func newQRCodeEvent(qrCode *qrcode.QRCode) *qrCodeEvent {
	event := &qrCodeEvent{qrCode: qrCode}
	event.SetEventNow()
	return event
}

func (m *Model) generateQRCode(fingerprint string) tview.Command {
	return tview.EventCommand(func() tcell.Event {
		content := "https://discord.com/ra/" + fingerprint
		qrCode, err := qrcode.New(content, qrcode.Low)
		if err != nil {
			return tcell.NewEventError(err)
		}
		qrCode.DisableBorder = true
		return newQRCodeEvent(qrCode)
	})
}

type userEvent struct {
	tcell.EventTime
	discriminator string
	username      string
}

func newUserEvent(discriminator, username string) *userEvent {
	event := &userEvent{discriminator: discriminator, username: username}
	event.SetEventNow()
	return event
}

func (m *Model) decryptUserPayload(encryptedPayload string) tview.Command {
	return tview.EventCommand(func() tcell.Event {
		decodedPayload, err := base64.StdEncoding.DecodeString(encryptedPayload)
		if err != nil {
			return tcell.NewEventError(err)
		}

		decryptedPayload, err := rsa.DecryptOAEP(sha256.New(), nil, m.privateKey, decodedPayload, nil)
		if err != nil {
			return tcell.NewEventError(err)
		}

		parts := strings.Split(string(decryptedPayload), ":")
		if len(parts) != 4 {
			return tcell.NewEventError(errors.New("invalid user payload"))
		}

		return newUserEvent(parts[1], parts[3])
	})
}

func (m *Model) exchangeTicket(ticket string) tview.Command {
	return tview.EventCommand(func() tcell.Event {
		headers := http.Headers()
		headers.Set("Referer", "https://discord.com/login")
		if m.fingerprint != "" {
			headers.Set("X-Fingerprint", m.fingerprint)
		}

		client := http.NewClient("")
		client.OnRequest = append(client.OnRequest, httputil.WithHeaders(headers))

		encryptedToken, err := client.ExchangeRemoteAuthTicket(ticket)
		if err != nil {
			return tcell.NewEventError(err)
		}

		decodedToken, err := base64.StdEncoding.DecodeString(encryptedToken)
		if err != nil {
			return tcell.NewEventError(err)
		}

		decryptedToken, err := rsa.DecryptOAEP(sha256.New(), nil, m.privateKey, decodedToken, nil)
		if err != nil {
			return tcell.NewEventError(err)
		}
		return newTokenEvent(string(decryptedToken))
	})
}
