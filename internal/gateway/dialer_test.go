//go:build !no_spoof_tls_fingerprint

package gateway

import (
	"testing"

	"github.com/gorilla/websocket"
)

func TestNewDialer(t *testing.T) {
	dialer := NewDialer()
	if dialer.NetDialTLSContext == nil {
		t.Error("TLS dialer is nil")
	}
	if dialer.Proxy != nil {
		t.Error("proxy is enabled")
	}
	if !dialer.EnableCompression {
		t.Error("compression is disabled")
	}
	if websocket.DefaultDialer.NetDialTLSContext != nil {
		t.Error("default dialer was modified")
	}
}
