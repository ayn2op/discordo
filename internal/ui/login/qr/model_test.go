package qr

import (
	"strings"
	"testing"

	"github.com/skip2/go-qrcode"
)

func TestModelSetRect(t *testing.T) {
	t.Run("renders content", func(t *testing.T) {
		m := NewModel()
		code, err := qrcode.New("test", qrcode.Low)
		if err != nil {
			t.Fatal(err)
		}
		m.qrCode = code
		m.SetRect(0, 0, 80, 24)
		text := m.Text()
		if !strings.Contains(text, m.status) || !strings.ContainsAny(text, "█▀▄") {
			t.Fatal("missing content")
		}
	})
}
