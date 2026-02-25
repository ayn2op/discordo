package login

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	stdhttp "net/http"
	"strings"
	"time"

	"github.com/ayn2op/discordo/internal/config"
	apphttp "github.com/ayn2op/discordo/internal/http"
	"github.com/ayn2op/discordo/internal/ui"
	"github.com/ayn2op/tview"
	"github.com/diamondburned/arikawa/v3/api"
	"github.com/gdamore/tcell/v3"
	"github.com/gorilla/websocket"
	"github.com/skip2/go-qrcode"
)

const gatewayURL = "wss://remote-auth-gateway.discord.gg/?v=2"

var endpointRemoteAuthLogin = api.EndpointMe + "/remote-auth/login"

type qrLogin struct {
	*tview.TextView
	app         *tview.Application
	cfg         *config.Config
	done        func(token string, err error)
	conn        *websocket.Conn
	privKey     *rsa.PrivateKey
	cancel      context.CancelFunc
	fingerprint string
}

func newQRLogin(app *tview.Application, cfg *config.Config, done func(token string, err error)) *qrLogin {
	q := &qrLogin{
		TextView: tview.NewTextView(),
		app:      app,
		cfg:      cfg,
		done:     done,
	}
	q.Box = ui.ConfigureBox(q.Box, &cfg.Theme)

	q.
		SetScrollable(true).
		SetWrap(false).
		SetTextAlign(tview.AlignmentCenter).
		SetChangedFunc(func() {
			q.app.QueueUpdateDraw(func() {})
		}).
		SetTitle("Login with QR")

	return q
}

func (q *qrLogin) InputHandler(event *tcell.EventKey, setFocus func(p tview.Primitive)) {
	if event.Key() == tcell.KeyEsc {
		q.stop()
		if q.done != nil {
			q.done("", nil)
		}
		return
	}
	q.TextView.InputHandler(event, setFocus)
}

func (q *qrLogin) start() {
	ctx, cancel := context.WithCancel(context.Background())
	q.cancel = cancel
	go q.run(ctx)
}

func (q *qrLogin) stop() {
	if q.cancel != nil {
		q.cancel()
	}
	if q.conn != nil {
		q.conn.Close()
	}
}

func (q *qrLogin) setText(s string) {
	builder := tview.NewLineBuilder()
	builder.Write(s, tcell.StyleDefault)
	q.setLines(builder.Finish())
}

func (q *qrLogin) setLines(lines []tview.Line) {
	q.app.QueueUpdateDraw(func() {
		q.SetLines(q.centerLines(lines))
	})
}

func (q *qrLogin) centerLines(lines []tview.Line) []tview.Line {
	_, _, _, height := q.GetInnerRect()
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

func (q *qrLogin) writeJSON(data any) error {
	return q.conn.WriteJSON(data)
}

type raHello struct {
	TimeoutMs         int `json:"timeout_ms"`
	HeartbeatInterval int `json:"heartbeat_interval"`
}

type raNonceProof struct {
	EncryptedNonce string `json:"encrypted_nonce"`
}

type raPendingInit struct {
	Fingerprint string `json:"fingerprint"`
}

type raPendingLogin struct {
	Ticket string `json:"ticket"`
}

type raPendingTicket struct {
	EncryptedUserPayload string `json:"encrypted_user_payload"`
}

func (q *qrLogin) run(ctx context.Context) {
	defer q.stop()
	q.setText("Preparing QR code...\n\nPress Esc to cancel")

	privKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		q.fail(err)
		return
	}
	q.privKey = privKey

	pubDER, err := x509.MarshalPKIXPublicKey(&privKey.PublicKey)
	if err != nil {
		q.fail(err)
		return
	}
	encodedPublicKey := base64.StdEncoding.EncodeToString(pubDER)

	headers := stdhttp.Header{}
	headers.Set("User-Agent", apphttp.BrowserUserAgent)
	headers.Set("Origin", "https://discord.com")

	q.setText("Connecting to Remote Auth Gateway...\n\nPress Esc to cancel")

	dialer := websocket.Dialer{
		Proxy:             stdhttp.ProxyFromEnvironment,
		HandshakeTimeout:  15 * time.Second,
		EnableCompression: true,
	}

	conn, resp, err := dialer.DialContext(ctx, gatewayURL, headers)
	if err != nil {
		var body []byte
		if resp != nil && resp.Body != nil {
			body, _ = io.ReadAll(resp.Body)
		}
		status := ""
		if resp != nil {
			status = resp.Status
		}
		q.fail(fmt.Errorf("websocket dial failed: %w, status=%s, body=%s", err, status, string(body)))
		return
	}
	q.conn = conn

	readCh := make(chan []byte, 1)
	readErr := make(chan error, 1)
	go func() {
		for {
			_, data, err := conn.ReadMessage()
			if err != nil {
				readErr <- err
				return
			}
			readCh <- data
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case err := <-readErr:
			if mapped := mapWSCloseError(err); mapped == nil {
				return
			} else {
				q.fail(mapped)
			}
			return
		case data := <-readCh:
			var opOnly struct {
				Op string `json:"op"`
			}
			if err := json.Unmarshal(data, &opOnly); err != nil {
				q.setText("Bad JSON: " + err.Error())
				q.fail(err)
				return
			}

			switch opOnly.Op {
			case "hello":
				var h raHello
				if err := json.Unmarshal(data, &h); err != nil {
					q.setText("Hello decode failed: " + err.Error())
					q.fail(err)
					return
				}
				if h.HeartbeatInterval > 0 {
					heartbeatTicker := time.NewTicker(time.Duration(h.HeartbeatInterval) * time.Millisecond)
					go func() {
						defer heartbeatTicker.Stop()
						for {
							select {
							case <-ctx.Done():
								return
							case <-heartbeatTicker.C:
								q.writeJSON(map[string]any{"op": "heartbeat"})
							}
						}
					}()
				}
				q.setText("Connected. Handshaking...\n\nPress Esc to cancel")
				if err := q.writeJSON(map[string]any{
					"op":                 "init",
					"encoded_public_key": encodedPublicKey,
				}); err != nil {
					q.setText("Init send failed: " + err.Error())
					q.fail(err)
					return
				}
			case "nonce_proof":
				var n raNonceProof
				if err := json.Unmarshal(data, &n); err != nil {
					q.setText("Nonce decode failed: " + err.Error())
					q.fail(err)
					return
				}
				enc, err := base64.StdEncoding.DecodeString(n.EncryptedNonce)
				if err != nil {
					q.setText("Nonce b64 decode failed: " + err.Error())
					q.fail(err)
					return
				}
				pt, err := rsa.DecryptOAEP(sha256.New(), rand.Reader, q.privKey, enc, nil)
				if err != nil {
					q.setText("Nonce decrypt failed: " + err.Error())
					q.fail(err)
					return
				}
				nonce := base64.RawURLEncoding.EncodeToString(pt)
				if err := q.writeJSON(map[string]any{"op": "nonce_proof", "nonce": nonce}); err != nil {
					q.setText("Nonce send failed: " + err.Error())
					q.fail(err)
					return
				}
			case "pending_remote_init":
				var p raPendingInit
				if err := json.Unmarshal(data, &p); err != nil {
					q.setText("Init decode failed: " + err.Error())
					q.fail(err)
					return
				}
				q.fingerprint = p.Fingerprint
				content := "https://discord.com/ra/" + p.Fingerprint
				qrLines, err := renderQR(content)
				if err != nil {
					q.setText("QR render failed: " + err.Error())
					q.fail(err)
					return
				}
				builder := tview.NewLineBuilder()
				builder.AppendLines(qrLines)
				builder.Write("\n\nScan with Discord mobile app\n\nPress Esc to cancel", tcell.StyleDefault)
				q.setLines(builder.Finish())
			case "heartbeat_ack":
			case "pending_ticket":
				var t raPendingTicket
				if err := json.Unmarshal(data, &t); err != nil {
					q.setText("Ticket decode failed: " + err.Error())
					q.fail(err)
					return
				}
				payload, err := base64.StdEncoding.DecodeString(t.EncryptedUserPayload)
				if err != nil {
					q.setText("Ticket payload b64 failed: " + err.Error())
					q.fail(err)
					return
				}
				pt, err := rsa.DecryptOAEP(sha256.New(), rand.Reader, q.privKey, payload, nil)
				if err != nil {
					q.setText("Ticket payload decrypt failed: " + err.Error())
					q.fail(err)
					return
				}
				parts := strings.SplitN(string(pt), ":", 4)
				var discriminator, username string
				if len(parts) == 4 {
					discriminator = parts[1]
					username = parts[3]
				}
				if discriminator == "" && username == "" {
					q.setText("Scan received.\n\nWaiting for approval on mobile...\n\nPress Esc to cancel")
				} else {
					q.setText("Logging in as " + username + "#" + discriminator + "\n\nConfirm on mobile...\n\nPress Esc to cancel")
				}
			case "pending_login":
				var p raPendingLogin
				if err := json.Unmarshal(data, &p); err != nil {
					q.setText("Login decode failed: " + err.Error())
					q.fail(err)
					return
				}
				q.setText("Authenticating...\n\nPlease wait")
				token, err := exchangeTicket(ctx, p.Ticket, q.fingerprint, q.privKey)
				if err != nil {
					q.setText("Ticket exchange failed: " + err.Error())
					q.fail(err)
					return
				}
				q.success(token)
				return
			case "cancel":
				q.setText("Login canceled on mobile")
				if q.done != nil {
					q.done("", nil)
				}
				return
			default:
			}
		}
	}
}

func renderQR(content string) ([]tview.Line, error) {
	code, err := qrcode.New(content, qrcode.Low)
	if err != nil {
		return nil, err
	}
	bitmap := code.Bitmap()
	builder := tview.NewLineBuilder()
	style := tcell.StyleDefault
	for y := 0; y < len(bitmap); y += 2 {
		for x := range bitmap[y] {
			top := bitmap[y][x]
			bottom := false
			if y+1 < len(bitmap) {
				bottom = bitmap[y+1][x]
			}
			if top && bottom {
				builder.Write("█", style)
			} else if top && !bottom {
				builder.Write("▀", style)
			} else if !top && bottom {
				builder.Write("▄", style)
			} else {
				builder.Write(" ", style)
			}
		}
		builder.NewLine()
	}
	return builder.Finish(), nil
}

func exchangeTicket(ctx context.Context, ticket string, fingerprint string, priv *rsa.PrivateKey) (string, error) {
	if ticket == "" {
		return "", errors.New("empty ticket")
	}
	body := map[string]string{"ticket": ticket}
	raw, err := json.Marshal(body)
	if err != nil {
		return "", err
	}
	req, err := stdhttp.NewRequestWithContext(ctx, stdhttp.MethodPost, endpointRemoteAuthLogin, bytes.NewReader(raw))
	if err != nil {
		return "", err
	}

	req.Header = apphttp.Headers()
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", apphttp.BrowserUserAgent)
	if fingerprint != "" {
		req.Header.Set("X-Fingerprint", fingerprint)
		req.Header.Set("Referer", "https://discord.com/ra/"+fingerprint)
	}

	client := &stdhttp.Client{Transport: apphttp.NewTransport(), Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("remote-auth login failed: %s: %s", resp.Status, string(b))
	}

	decoder := json.NewDecoder(resp.Body)

	var out struct {
		EncryptedToken string `json:"encrypted_token"`
	}
	if err := decoder.Decode(&out); err != nil {
		return "", err
	}
	if out.EncryptedToken == "" {
		return "", fmt.Errorf("no encrypted_token in response")
	}
	enc, err := base64.StdEncoding.DecodeString(out.EncryptedToken)
	if err != nil {
		return "", err
	}
	pt, err := rsa.DecryptOAEP(sha256.New(), rand.Reader, priv, enc, nil)
	if err != nil {
		return "", err
	}
	return string(pt), nil
}

func (q *qrLogin) success(token string) {
	if q.done != nil {
		q.done(token, nil)
	}
}

func (q *qrLogin) fail(err error) {
	slog.Error("qr login failed", "err", err)
	if q.done != nil {
		q.done("", err)
	}
}

func mapWSCloseError(err error) error {
	var cerr *websocket.CloseError
	if errors.As(err, &cerr) {
		switch cerr.Code {
		case 1000:
			return errors.New("session closed")
		case 4000:
			return errors.New("remote auth: invalid version")
		case 4001:
			return errors.New("remote auth: decode error")
		case 4002:
			return errors.New("remote auth: handshake failure")
		case 4003:
			return errors.New("remote auth: session timed out")
		}
	}
	return err
}
