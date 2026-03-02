package clipboard

import (
	"bytes"
	"log/slog"
	"os"
	"os/exec"

	nativeclipboard "github.com/aymanbagabas/go-nativeclipboard"
)

// Format represents the type of clipboard content.
type Format int

const (
	FmtText  Format = iota // plain text
	FmtImage               // image data
)

var wayland bool

func Init() error {
	if _, ok := os.LookupEnv("WAYLAND_DISPLAY"); !ok {
		return nil
	}
	if _, err := exec.LookPath("wl-copy"); err != nil {
		return nil
	}
	if _, err := exec.LookPath("wl-paste"); err != nil {
		return nil
	}
	wayland = true
	return nil
}

func Read(t Format) []byte {
	if wayland {
		return waylandRead(t)
	}
	f := formatToNative(t)
	data, err := f.Read()
	if err != nil {
		slog.Error("failed to read clipboard", "err", err)
		return nil
	}
	return data
}

func Write(t Format, buf []byte) {
	if wayland {
		waylandWrite(t, buf)
		return
	}
	f := formatToNative(t)
	if _, err := f.Write(buf); err != nil {
		slog.Error("failed to write to clipboard", "err", err)
	}
}

func waylandRead(t Format) []byte {
	cmd := exec.Command("wl-paste", "-nt", waylandMIME(t))
	out, err := cmd.Output()
	if err != nil {
		slog.Error("failed to read clipboard via wl-paste", "err", err)
		return nil
	}
	return out
}

func waylandWrite(t Format, buf []byte) {
	cmd := exec.Command("wl-copy", "-t", waylandMIME(t))
	cmd.Stdin = bytes.NewReader(buf)
	if err := cmd.Run(); err != nil {
		slog.Error("failed to write to clipboard via wl-copy", "err", err)
	}
}

func waylandMIME(t Format) string {
	switch t {
	case FmtImage:
		return "image/png"
	default:
		return "text/plain;charset=utf-8"
	}
}

func formatToNative(t Format) nativeclipboard.Format {
	switch t {
	case FmtImage:
		return nativeclipboard.Image
	default:
		return nativeclipboard.Text
	}
}
