package tls

import (
	"testing"

	"github.com/bogdanfinn/tls-client/profiles"
)

func TestBrowserIdentity(t *testing.T) {
	hello := profiles.Chrome_146.GetClientHelloId()

	tests := []struct {
		name string
		got  string
		want string
	}{
		{"name", BrowserName(), hello.Client},
		{"version", BrowserVersion(), hello.Version + ".0.0.0"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.got != test.want {
				t.Fatalf("got = %q, want = %q", test.got, test.want)
			}
		})
	}
}
