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

	tea "charm.land/bubbletea/v2"
	"github.com/ayn2op/discordo/internal/http"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/utils/httputil"
	"github.com/gorilla/websocket"
	"github.com/skip2/go-qrcode"
)

// https://docs.discord.food/remote-authentication/desktop#endpoint
const remoteAuthGatewayURL = "wss://remote-auth-gateway.discord.gg/?v=2"

type (
	connCreateMsg *websocket.Conn
	connCloseMsg  struct{}
)

func (m Model) connect() tea.Cmd {
	return func() tea.Msg {
		conn, _, err := websocket.DefaultDialer.Dial(remoteAuthGatewayURL, http.Headers())
		if err != nil {
			return err
		}
		return connCreateMsg(conn)
	}
}

func (m Model) close() tea.Cmd {
	return func() tea.Msg {
		if m.conn != nil {
			if err := m.conn.Close(); err != nil {
				return err
			}
		}
		return connCloseMsg{}
	}
}

type (
	helloMsg struct {
		// HeartbeatInterval is the minimum interval (in milliseconds) the client should heartbeat at.
		HeartbeatInterval int `json:"heartbeat_interval"`
		TimeoutMS         int `json:"timeout_ms"`
	}

	nonceProofMsg struct {
		// EncryptedNonce is the base64-encoded nonce encrypted with the client's public key.
		EncryptedNonce string `json:"encrypted_nonce"`
	}

	pendingRemoteInitMsg struct {
		Fingerprint string `json:"fingerprint"`
	}

	pendingTicketMsg struct {
		EncryptedUserPayload string `json:"encrypted_user_payload"`
	}

	pendingLoginMsg struct {
		Ticket string `json:"ticket"`
	}

	cancelMsg struct{}
)

func (m Model) listen() tea.Cmd {
	return func() tea.Msg {
		if m.conn == nil {
			return nil
		}

		_, data, err := m.conn.ReadMessage()
		if err != nil {
			return err
		}

		var payload struct {
			Op string `json:"op"`
		}
		if err := json.Unmarshal(data, &payload); err != nil {
			return err
		}

		switch payload.Op {
		case "hello":
			var msg helloMsg
			if err := json.Unmarshal(data, &msg); err != nil {
				return err
			}
			return msg
		case "nonce_proof":
			var msg nonceProofMsg
			if err := json.Unmarshal(data, &msg); err != nil {
				return err
			}
			return msg
		case "pending_remote_init":
			var msg pendingRemoteInitMsg
			if err := json.Unmarshal(data, &msg); err != nil {
				return err
			}
			return msg
		case "pending_ticket":
			var msg pendingTicketMsg
			if err := json.Unmarshal(data, &msg); err != nil {
				return err
			}
			return msg
		case "cancel":
			var msg cancelMsg
			if err := json.Unmarshal(data, &msg); err != nil {
				return err
			}
			return msg
		case "pending_login":
			var msg pendingLoginMsg
			if err := json.Unmarshal(data, &msg); err != nil {
				return err
			}
			return msg
		default:
			return nil
		}
	}
}

type heartbeatTickMsg struct{}

func (m Model) heartbeat() tea.Cmd {
	return tea.Tick(m.heartbeatInterval, func(t time.Time) tea.Msg {
		return heartbeatTickMsg{}
	})
}

func (m Model) sendHeartbeat() tea.Cmd {
	return func() tea.Msg {
		if m.conn == nil {
			return nil
		}
		data := struct {
			Op string `json:"op"`
		}{"heartbeat"}
		return m.conn.WriteJSON(data)
	}
}

type privateKeyMsg *rsa.PrivateKey

func (m Model) generatePrivateKey() tea.Cmd {
	return func() tea.Msg {
		privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			return err
		}
		return privateKeyMsg(privateKey)
	}
}

// https://docs.discord.food/remote-authentication/desktop#handshaking
func (m Model) sendInit() tea.Cmd {
	return func() tea.Msg {
		spki, err := x509.MarshalPKIXPublicKey(m.privateKey.Public())
		if err != nil {
			return err
		}
		encodedPublicKey := base64.StdEncoding.EncodeToString(spki)
		data := struct {
			Op string `json:"op"`
			// EncodedPublicKey is the base64-encoded SPKI of the client's 2048-bit RSA-OAEP public key.
			EncodedPublicKey string `json:"encoded_public_key"`
		}{"init", encodedPublicKey}
		return m.conn.WriteJSON(data)
	}
}

func (m Model) sendNonceProof(encryptedNonce string) tea.Cmd {
	return func() tea.Msg {
		decodedNonce, err := base64.StdEncoding.DecodeString(encryptedNonce)
		if err != nil {
			return err
		}

		decryptedNonce, err := rsa.DecryptOAEP(sha256.New(), nil, m.privateKey, decodedNonce, nil)
		if err != nil {
			return err
		}

		encodedNonce := base64.RawURLEncoding.EncodeToString(decryptedNonce)
		data := struct {
			Op string `json:"op"`
			// Nonce is the base64URL-encoded decrypted nonce.
			Nonce string `json:"nonce"`
		}{"nonce_proof", encodedNonce}
		return m.conn.WriteJSON(data)
	}
}

type qrCodeMsg *qrcode.QRCode

func (m Model) generateQRCode(fingerprint string) tea.Cmd {
	return func() tea.Msg {
		content := "https://discord.com/ra/" + fingerprint
		qrCode, err := qrcode.New(content, qrcode.Low)
		if err != nil {
			return err
		}
		qrCode.DisableBorder = true
		return qrCodeMsg(qrCode)
	}
}

type userMsg struct {
	ID            discord.UserID
	Discriminator string
	Avatar        string
	Username      string
}

func (m Model) decryptUserPayload(encryptedPayload string) tea.Cmd {
	return func() tea.Msg {
		decodedPayload, err := base64.StdEncoding.DecodeString(encryptedPayload)
		if err != nil {
			return err
		}

		decryptedPayload, err := rsa.DecryptOAEP(sha256.New(), nil, m.privateKey, decodedPayload, nil)
		if err != nil {
			return err
		}

		parts := strings.Split(string(decryptedPayload), ":")
		if len(parts) != 4 {
			return errors.New("invalid user payload")
		}

		id, err := discord.ParseSnowflake(parts[0])
		if err != nil {
			return err
		}
		return userMsg{
			ID:            discord.UserID(id),
			Discriminator: parts[1],
			Avatar:        parts[2],
			Username:      parts[3],
		}

	}
}

type TokenMsg string

func (m Model) exchangeTicket(ticket string) tea.Cmd {
	return func() tea.Msg {
		headers := http.Headers()
		headers.Set("Referer", "https://discord.com/login")
		if m.fingerprint != "" {
			headers.Set("X-Fingerprint", m.fingerprint)
		}

		// Create an API client without a token.
		client := http.NewClient("")
		client.OnRequest = append(client.OnRequest, httputil.WithHeaders(headers))

		encryptedToken, err := client.ExchangeRemoteAuthTicket(ticket)
		if err != nil {
			return err
		}

		decodedToken, err := base64.StdEncoding.DecodeString(encryptedToken)
		if err != nil {
			return err
		}

		decryptedToken, err := rsa.DecryptOAEP(sha256.New(), nil, m.privateKey, decodedToken, nil)
		if err != nil {
			return err
		}
		return TokenMsg(decryptedToken)
	}
}
