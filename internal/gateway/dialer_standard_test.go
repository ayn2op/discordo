//go:build no_spoof_tls_fingerprint

package gateway

import "testing"

func TestNewDialer(t *testing.T) {
	dialer := NewDialer()
	t.Run("standard TLS", func(t *testing.T) {
		if dialer.NetDialTLSContext != nil {
			t.Error("custom TLS dialer is enabled")
		}
	})
	t.Run("environment proxy", func(t *testing.T) {
		if dialer.Proxy == nil {
			t.Error("environment proxy is disabled")
		}
	})
	t.Run("compression", func(t *testing.T) {
		if !dialer.EnableCompression {
			t.Error("compression is disabled")
		}
	})
}
