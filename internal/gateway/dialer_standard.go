//go:build no_spoof_tls_fingerprint

package gateway

import "github.com/gorilla/websocket"

func NewDialer() websocket.Dialer {
	dialer := *websocket.DefaultDialer
	dialer.EnableCompression = true
	return dialer
}
