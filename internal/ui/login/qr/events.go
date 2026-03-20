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

const remoteAuthGatewayURL = "wss://remote-auth-gateway.discord.gg/?v=2"

type connCreateEvent struct {
	tcell.EventTime
	conn *websocket.Conn
}

type connCloseEvent struct{ tcell.EventTime }

func (m *Model) connect() tview.Command {
	return func() tview.Event {
		headers := http.Headers()
		headers.Set("User-Agent", http.BrowserUserAgent)
		conn, _, err := websocket.DefaultDialer.Dial(remoteAuthGatewayURL, headers)
		if err != nil {
			return tcell.NewEventError(err)
		}
		return &connCreateEvent{conn: conn}
	}
}

func (m *Model) close() tview.Command {
	return func() tview.Event {
		if m.conn != nil {
			if err := m.conn.Close(); err != nil {
				return tcell.NewEventError(err)
			}
		}
		return &connCloseEvent{}
	}
}

type helloEvent struct {
	tcell.EventTime
	heartbeatInterval int
	timeoutMS         int
}

type nonceProofEvent struct {
	tcell.EventTime
	encryptedNonce string
}

type pendingRemoteInitEvent struct {
	tcell.EventTime
	fingerprint string
}

type pendingTicketEvent struct {
	tcell.EventTime
	encryptedUserPayload string
}

type pendingLoginEvent struct {
	tcell.EventTime
	ticket string
}

type cancelEvent struct{ tcell.EventTime }

func (m *Model) listen() tview.Command {
	return func() tview.Event {
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
			return &helloEvent{heartbeatInterval: payload.HeartbeatInterval, timeoutMS: payload.TimeoutMS}
		case "nonce_proof":
			var payload struct {
				EncryptedNonce string `json:"encrypted_nonce"`
			}
			if err := json.Unmarshal(data, &payload); err != nil {
				return tcell.NewEventError(err)
			}
			return &nonceProofEvent{encryptedNonce: payload.EncryptedNonce}
		case "pending_remote_init":
			var payload struct {
				Fingerprint string `json:"fingerprint"`
			}
			if err := json.Unmarshal(data, &payload); err != nil {
				return tcell.NewEventError(err)
			}
			return &pendingRemoteInitEvent{fingerprint: payload.Fingerprint}
		case "pending_ticket":
			var payload struct {
				EncryptedUserPayload string `json:"encrypted_user_payload"`
			}
			if err := json.Unmarshal(data, &payload); err != nil {
				return tcell.NewEventError(err)
			}
			return &pendingTicketEvent{encryptedUserPayload: payload.EncryptedUserPayload}
		case "cancel":
			return &cancelEvent{}
		case "pending_login":
			var payload struct {
				Ticket string `json:"ticket"`
			}
			if err := json.Unmarshal(data, &payload); err != nil {
				return tcell.NewEventError(err)
			}
			return &pendingLoginEvent{ticket: payload.Ticket}
		default:
			return nil
		}
	}
}

type heartbeatTickEvent struct{ tcell.EventTime }

func (m *Model) heartbeat() tview.Command {
	return func() tview.Event {
		time.Sleep(m.heartbeatInterval)
		return &heartbeatTickEvent{}
	}
}

func (m *Model) sendHeartbeat() tview.Command {
	return func() tview.Event {
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
	}
}

type privateKeyEvent struct {
	tcell.EventTime
	privateKey *rsa.PrivateKey
}

func (m *Model) generatePrivateKey() tview.Command {
	return func() tview.Event {
		privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			return tcell.NewEventError(err)
		}
		return &privateKeyEvent{privateKey: privateKey}
	}
}

func (m *Model) sendInit() tview.Command {
	return func() tview.Event {
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
	}
}

func (m *Model) sendNonceProof(encryptedNonce string) tview.Command {
	return func() tview.Event {
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
	}
}

type qrCodeEvent struct {
	tcell.EventTime
	qrCode *qrcode.QRCode
}

func (m *Model) generateQRCode(fingerprint string) tview.Command {
	return func() tview.Event {
		content := "https://discord.com/ra/" + fingerprint
		qrCode, err := qrcode.New(content, qrcode.Low)
		if err != nil {
			return tcell.NewEventError(err)
		}
		qrCode.DisableBorder = true
		return &qrCodeEvent{qrCode: qrCode}
	}
}

type userEvent struct {
	tcell.EventTime
	discriminator string
	username      string
}

func (m *Model) decryptUserPayload(encryptedPayload string) tview.Command {
	return func() tview.Event {
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

		return &userEvent{discriminator: parts[1], username: parts[3]}
	}
}

func (m *Model) exchangeTicket(ticket string) tview.Command {
	return func() tview.Event {
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
		return &TokenEvent{Token: string(decryptedToken)}
	}
}
