package tls

import "testing"

func TestBrowserName(t *testing.T) {
	want := clientProfile.GetClientHelloId().Client
	if got := BrowserName(); got != want {
		t.Fatalf("BrowserName() = %q, want %q", got, want)
	}
}

func TestBrowserVersion(t *testing.T) {
	want := clientProfile.GetClientHelloId().Version + ".0.0.0"
	if got := BrowserVersion(); got != want {
		t.Fatalf("BrowserVersion() = %q, want %q", got, want)
	}
}
