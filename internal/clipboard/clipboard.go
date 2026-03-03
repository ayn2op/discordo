package clipboard

import (
	"bytes"
	"os"
	"os/exec"

	nativeclipboard "github.com/aymanbagabas/go-nativeclipboard"
)

// Format represents the type of clipboard content.
type Format = nativeclipboard.Format

const (
	FmtText  = nativeclipboard.Text
	FmtImage = nativeclipboard.Image
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

func Read(t Format) ([]byte, error) {
	if wayland {
		return waylandRead(t)
	}
	return t.Read()
}

func Write(t Format, buf []byte) error {
	if wayland {
		return waylandWrite(t, buf)
	}
	_, err := t.Write(buf)
	return err
}

func waylandRead(t Format) ([]byte, error) {
	return exec.Command("wl-paste", "-nt", waylandMIME(t)).Output()
}

func waylandWrite(t Format, buf []byte) error {
	cmd := exec.Command("wl-copy", "-t", waylandMIME(t))
	cmd.Stdin = bytes.NewReader(buf)
	return cmd.Run()
}

func waylandMIME(t Format) string {
	switch t {
	case FmtImage:
		return "image/png"
	default:
		return "text/plain;charset=utf-8"
	}
}

