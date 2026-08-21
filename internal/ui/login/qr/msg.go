package qr

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	json "encoding/json/v2"
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

func connect() tview.Cmd {
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

func closeConn(conn *websocket.Conn) tview.Cmd {
	return func() tview.Msg {
		if conn != nil {
			if err := conn.Close(); err != nil {
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

func listen(conn *websocket.Conn) tview.Cmd {
	return func() tview.Msg {
		if conn == nil {
			return nil
		}

		_, data, err := conn.ReadMessage()
		if err != nil {
			return errMsg(err)
		}

		return decodeMessage(data)
	}
}

func decodeMessage(data []byte) tview.Msg {
	var payload struct {
		Op                   string `json:"op"`
		HeartbeatInterval    int    `json:"heartbeat_interval"`
		TimeoutMS            int    `json:"timeout_ms"`
		EncryptedNonce       string `json:"encrypted_nonce"`
		Fingerprint          string `json:"fingerprint"`
		EncryptedUserPayload string `json:"encrypted_user_payload"`
		Ticket               string `json:"ticket"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		return errMsg(err)
	}

	switch payload.Op {
	case "hello":
		return helloMsg{heartbeatInterval: payload.HeartbeatInterval, timeoutMS: payload.TimeoutMS}
	case "nonce_proof":
		return nonceProofMsg{encryptedNonce: payload.EncryptedNonce}
	case "pending_remote_init":
		return pendingRemoteInitMsg{fingerprint: payload.Fingerprint}
	case "pending_ticket":
		return pendingTicketMsg{encryptedUserPayload: payload.EncryptedUserPayload}
	case "cancel":
		return cancelMsg{}
	case "pending_login":
		return pendingLoginMsg{ticket: payload.Ticket}
	default:
		return nil
	}
}

type heartbeatTickMsg struct{}

func heartbeat(interval time.Duration) tview.Cmd {
	return func() tview.Msg {
		time.Sleep(interval)
		return heartbeatTickMsg{}
	}
}

func sendHeartbeat(conn *websocket.Conn) tview.Cmd {
	return func() tview.Msg {
		if conn == nil {
			return nil
		}
		data := struct {
			Op string `json:"op"`
		}{"heartbeat"}
		if err := conn.WriteJSON(data); err != nil {
			return errMsg(err)
		}
		return nil
	}
}

type privateKeyMsg struct {
	privateKey *rsa.PrivateKey
}

func generatePrivateKey() tview.Cmd {
	return func() tview.Msg {
		privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			return errMsg(err)
		}
		return privateKeyMsg{privateKey: privateKey}
	}
}

func sendInit(conn *websocket.Conn, privateKey *rsa.PrivateKey) tview.Cmd {
	return func() tview.Msg {
		if privateKey == nil {
			return errMsg(errors.New("missing private key"))
		}
		spki, err := x509.MarshalPKIXPublicKey(privateKey.Public())
		if err != nil {
			return errMsg(err)
		}
		encodedPublicKey := base64.StdEncoding.EncodeToString(spki)
		data := struct {
			Op               string `json:"op"`
			EncodedPublicKey string `json:"encoded_public_key"`
		}{"init", encodedPublicKey}
		if err := conn.WriteJSON(data); err != nil {
			return errMsg(err)
		}
		return nil
	}
}

func sendNonceProof(conn *websocket.Conn, privateKey *rsa.PrivateKey, encryptedNonce string) tview.Cmd {
	return func() tview.Msg {
		decodedNonce, err := base64.StdEncoding.DecodeString(encryptedNonce)
		if err != nil {
			return errMsg(err)
		}

		decryptedNonce, err := rsa.DecryptOAEP(sha256.New(), nil, privateKey, decodedNonce, nil)
		if err != nil {
			return errMsg(err)
		}

		encodedNonce := base64.RawURLEncoding.EncodeToString(decryptedNonce)
		data := struct {
			Op    string `json:"op"`
			Nonce string `json:"nonce"`
		}{"nonce_proof", encodedNonce}
		if err := conn.WriteJSON(data); err != nil {
			return errMsg(err)
		}
		return nil
	}
}

type qrCodeMsg struct {
	qrCode *qrcode.QRCode
}

func generateQRCode(fingerprint string) tview.Cmd {
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

func decryptUserPayload(privateKey *rsa.PrivateKey, encryptedPayload string) tview.Cmd {
	return func() tview.Msg {
		decodedPayload, err := base64.StdEncoding.DecodeString(encryptedPayload)
		if err != nil {
			return errMsg(err)
		}

		decryptedPayload, err := rsa.DecryptOAEP(sha256.New(), nil, privateKey, decodedPayload, nil)
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

func exchangeTicket(fingerprint string, privateKey *rsa.PrivateKey, ticket string) tview.Cmd {
	return func() tview.Msg {
		headers := http.Headers()
		headers.Set("Referer", "https://discord.com/login")
		if fingerprint != "" {
			headers.Set("X-Fingerprint", fingerprint)
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

		decryptedToken, err := rsa.DecryptOAEP(sha256.New(), nil, privateKey, decodedToken, nil)
		if err != nil {
			return errMsg(err)
		}
		return TokenMsg(string(decryptedToken))
	}
}
