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

	"github.com/ayn2op/arikawa/v3/utils/httputil"
	"github.com/ayn2op/discordo/internal/gateway"
	"github.com/ayn2op/discordo/internal/http"
	"github.com/ayn2op/tview"
	"github.com/gorilla/websocket"
	"github.com/skip2/go-qrcode"
)

type (
	TokenMsg string
	errMsg   error
)

const remoteAuthGatewayURL = "wss://remote-auth-gateway.discord.gg/?v=2"

type connCreateMsg struct {
	conn *websocket.Conn
}

type connCloseMsg struct{}

func (m *Model) connect() tview.Cmd {
	return func() tview.Msg {
		headers := http.Headers()
		headers.Set("User-Agent", http.BrowserUserAgent())
		dialer := gateway.NewDialer()
		conn, _, err := dialer.Dial(remoteAuthGatewayURL, headers)
		if err != nil {
			return errMsg(err)
		}
		return connCreateMsg{conn: conn}
	}
}

func (m *Model) close() tview.Cmd {
	return func() tview.Msg {
		if m.conn != nil {
			if err := m.conn.Close(); err != nil {
				return errMsg(err)
			}
		}
		return connCloseMsg{}
	}
}

type helloMsg struct {
	heartbeatInterval int
	timeoutMS         int
}

type nonceProofMsg struct {
	encryptedNonce string
}

type pendingRemoteInitMsg struct {
	fingerprint string
}

type pendingTicketMsg struct {
	encryptedUserPayload string
}

type pendingLoginMsg struct {
	ticket string
}

type cancelMsg struct{}

func (m *Model) listen() tview.Cmd {
	return func() tview.Msg {
		if m.conn == nil {
			return nil
		}

		_, data, err := m.conn.ReadMessage()
		if err != nil {
			return errMsg(err)
		}

		var payload struct {
			Op string `json:"op"`
		}
		if err := json.Unmarshal(data, &payload); err != nil {
			return errMsg(err)
		}

		switch payload.Op {
		case "hello":
			var payload struct {
				HeartbeatInterval int `json:"heartbeat_interval"`
				TimeoutMS         int `json:"timeout_ms"`
			}
			if err := json.Unmarshal(data, &payload); err != nil {
				return errMsg(err)
			}
			return helloMsg{heartbeatInterval: payload.HeartbeatInterval, timeoutMS: payload.TimeoutMS}
		case "nonce_proof":
			var payload struct {
				EncryptedNonce string `json:"encrypted_nonce"`
			}
			if err := json.Unmarshal(data, &payload); err != nil {
				return errMsg(err)
			}
			return nonceProofMsg{encryptedNonce: payload.EncryptedNonce}
		case "pending_remote_init":
			var payload struct {
				Fingerprint string `json:"fingerprint"`
			}
			if err := json.Unmarshal(data, &payload); err != nil {
				return errMsg(err)
			}
			return pendingRemoteInitMsg{fingerprint: payload.Fingerprint}
		case "pending_ticket":
			var payload struct {
				EncryptedUserPayload string `json:"encrypted_user_payload"`
			}
			if err := json.Unmarshal(data, &payload); err != nil {
				return errMsg(err)
			}
			return pendingTicketMsg{encryptedUserPayload: payload.EncryptedUserPayload}
		case "cancel":
			return cancelMsg{}
		case "pending_login":
			var payload struct {
				Ticket string `json:"ticket"`
			}
			if err := json.Unmarshal(data, &payload); err != nil {
				return errMsg(err)
			}
			return pendingLoginMsg{ticket: payload.Ticket}
		default:
			return nil
		}
	}
}

type heartbeatTickMsg struct{}

func (m *Model) heartbeat() tview.Cmd {
	return func() tview.Msg {
		time.Sleep(m.heartbeatInterval)
		return heartbeatTickMsg{}
	}
}

func (m *Model) sendHeartbeat() tview.Cmd {
	return func() tview.Msg {
		if m.conn == nil {
			return nil
		}
		data := struct {
			Op string `json:"op"`
		}{"heartbeat"}
		if err := m.conn.WriteJSON(data); err != nil {
			return errMsg(err)
		}
		return nil
	}
}

type privateKeyMsg struct {
	privateKey *rsa.PrivateKey
}

func (m *Model) generatePrivateKey() tview.Cmd {
	return func() tview.Msg {
		privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			return errMsg(err)
		}
		return privateKeyMsg{privateKey: privateKey}
	}
}

func (m *Model) sendInit() tview.Cmd {
	return func() tview.Msg {
		if m.privateKey == nil {
			return errMsg(errors.New("missing private key"))
		}
		spki, err := x509.MarshalPKIXPublicKey(m.privateKey.Public())
		if err != nil {
			return errMsg(err)
		}
		encodedPublicKey := base64.StdEncoding.EncodeToString(spki)
		data := struct {
			Op               string `json:"op"`
			EncodedPublicKey string `json:"encoded_public_key"`
		}{"init", encodedPublicKey}
		if err := m.conn.WriteJSON(data); err != nil {
			return errMsg(err)
		}
		return nil
	}
}

func (m *Model) sendNonceProof(encryptedNonce string) tview.Cmd {
	return func() tview.Msg {
		decodedNonce, err := base64.StdEncoding.DecodeString(encryptedNonce)
		if err != nil {
			return errMsg(err)
		}

		decryptedNonce, err := rsa.DecryptOAEP(sha256.New(), nil, m.privateKey, decodedNonce, nil)
		if err != nil {
			return errMsg(err)
		}

		encodedNonce := base64.RawURLEncoding.EncodeToString(decryptedNonce)
		data := struct {
			Op    string `json:"op"`
			Nonce string `json:"nonce"`
		}{"nonce_proof", encodedNonce}
		if err := m.conn.WriteJSON(data); err != nil {
			return errMsg(err)
		}
		return nil
	}
}

type qrCodeMsg struct {
	qrCode *qrcode.QRCode
}

func (m *Model) generateQRCode(fingerprint string) tview.Cmd {
	return func() tview.Msg {
		content := "https://discord.com/ra/" + fingerprint
		qrCode, err := qrcode.New(content, qrcode.Low)
		if err != nil {
			return errMsg(err)
		}
		qrCode.DisableBorder = true
		return qrCodeMsg{qrCode: qrCode}
	}
}

type userMsg struct {
	discriminator string
	username      string
}

func (m *Model) decryptUserPayload(encryptedPayload string) tview.Cmd {
	return func() tview.Msg {
		decodedPayload, err := base64.StdEncoding.DecodeString(encryptedPayload)
		if err != nil {
			return errMsg(err)
		}

		decryptedPayload, err := rsa.DecryptOAEP(sha256.New(), nil, m.privateKey, decodedPayload, nil)
		if err != nil {
			return errMsg(err)
		}

		parts := strings.Split(string(decryptedPayload), ":")
		if len(parts) != 4 {
			return errMsg(errors.New("invalid user payload"))
		}

		return userMsg{discriminator: parts[1], username: parts[3]}
	}
}

func (m *Model) exchangeTicket(ticket string) tview.Cmd {
	return func() tview.Msg {
		headers := http.Headers()
		headers.Set("Referer", "https://discord.com/login")
		if m.fingerprint != "" {
			headers.Set("X-Fingerprint", m.fingerprint)
		}

		client := http.NewClient("")
		client.OnRequest = append(client.OnRequest, httputil.WithHeaders(headers))

		encryptedToken, err := client.ExchangeRemoteAuthTicket(ticket)
		if err != nil {
			return errMsg(err)
		}

		decodedToken, err := base64.StdEncoding.DecodeString(encryptedToken)
		if err != nil {
			return errMsg(err)
		}

		decryptedToken, err := rsa.DecryptOAEP(sha256.New(), nil, m.privateKey, decodedToken, nil)
		if err != nil {
			return errMsg(err)
		}
		return TokenMsg(string(decryptedToken))
	}
}
