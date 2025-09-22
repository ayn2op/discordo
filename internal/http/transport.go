package http

import (
	"compress/gzip"
	"io"
	"net/http"

	"github.com/andybalholm/brotli"
)

type Transport struct {
	base http.RoundTripper
}

func NewTransport() *Transport {
	return &Transport{
		base: http.DefaultTransport,
	}
}

func (t *Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := t.base.RoundTrip(req)
	if err != nil {
		return nil, err
	}

	switch resp.Header.Get("Content-Encoding") {
	case "gzip":
		resp.Body, err = gzip.NewReader(resp.Body)
		if err != nil {
			resp.Body.Close()
			return nil, err
		}

		resp.Header.Del("Content-Encoding")
	case "br":
		resp.Body = io.NopCloser(brotli.NewReader(resp.Body))
		resp.Header.Del("Content-Encoding")
	}

	return resp, nil
}
