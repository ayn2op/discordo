package http

import (
	"io"
	"net/http"

	"github.com/andybalholm/brotli"
	"github.com/klauspost/compress/gzhttp"
)

type Transport struct {
	base http.RoundTripper
}

func NewTransport() *Transport {
	return &Transport{
		base: gzhttp.Transport(http.DefaultTransport, gzhttp.TransportAlwaysDecompress(true)),
	}
}

func (t *Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := t.base.RoundTrip(req)
	if err != nil {
		return nil, err
	}

	if resp.Header.Get("Content-Encoding") == "br" {
		resp.Body = io.NopCloser(brotli.NewReader(resp.Body))
		resp.Header.Del("Content-Encoding")
		resp.Header.Del("Content-Length")
		resp.ContentLength = -1
		resp.Uncompressed = true
	}

	return resp, nil
}
