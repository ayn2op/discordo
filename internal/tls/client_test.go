//go:build !no_spoof_tls_fingerprint

package tls

import "testing"

func TestNewClient(t *testing.T) {
	client := NewClient()
	if client == nil {
		t.Fatal("client is nil")
	}
	if client.GetTLSDialer() == nil {
		t.Fatal("TLS dialer is nil")
	}
}
