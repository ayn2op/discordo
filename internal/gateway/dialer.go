//go:build !no_spoof_tls_fingerprint

package gateway

import (
	"github.com/ayn2op/discordo/internal/tls"
	tls_client "github.com/bogdanfinn/tls-client"
	"github.com/gorilla/websocket"
)

func NewDialer() websocket.Dialer {
	client := tls.NewClient(tls_client.WithForceHttp1())
	dialer := *websocket.DefaultDialer
	dialer.Proxy = nil
	dialer.EnableCompression = true
	dialer.NetDialTLSContext = client.GetTLSDialer()
	return dialer
}
