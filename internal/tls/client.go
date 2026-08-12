//go:build !no_spoof_tls_fingerprint

package tls

import (
	"fmt"

	tls_client "github.com/bogdanfinn/tls-client"
)

func NewClient(options ...tls_client.HttpClientOption) tls_client.HttpClient {
	options = append([]tls_client.HttpClientOption{tls_client.WithClientProfile(clientProfile), tls_client.WithRandomTLSExtensionOrder()}, options...)
	client, err := tls_client.NewHttpClient(tls_client.NewNoopLogger(), options...)
	if err != nil {
		panic(fmt.Errorf("failed to create new tls http client: %w", err))
	}
	return client
}
