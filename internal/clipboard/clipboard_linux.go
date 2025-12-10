//go:build linux

package clipboard

import (
	"bytes"
	designClipb "golang.design/x/clipboard"
	"log/slog"
	"os"
	"os/exec"
)

var wayland bool
var inited bool

func Init() error {
	if _, ok := os.LookupEnv("WAYLAND_DISPLAY"); !ok {
		inited = true
		return designClipb.Init()
	}
	if _, err := exec.LookPath("wl-copy"); err != nil {
		return err
	}
	if _, err := exec.LookPath("wl-paste"); err != nil {
		return err
	}
	wayland = true
	inited = true
	return nil
}

func Read(t Format) []byte {
	if !inited {
		return nil
	}
	if !wayland {
		return designClipb.Read(designClipb.Format(t))
	}
	// -n: Don't print a newline at the end
	// -t type: MIME type specifier
	cmd := exec.Command("wl-paste", "-nt", formatToType(t))
	outBuffer := bytes.Buffer{}
	cmd.Stdout = &outBuffer
	if err := cmd.Run(); err != nil {
		slog.Error("failed to read clipboard", "err", err)
		return nil
	}
	return outBuffer.Bytes()
}

func Write(t Format, buf []byte) <-chan struct{} {
	if !inited {
		return nil
	}
	if !wayland {
		return designClipb.Write(designClipb.Format(t), buf)
	}
	// -t type: MIME type specifier
	cmd := exec.Command("wl-copy", "-t", formatToType(t))
	cmd.Stdin = bytes.NewReader(buf)
	if err := cmd.Run(); err != nil {
		slog.Error("failed to write to clipboard", "err", err)
	}
	return nil
}

func formatToType(t Format) string {
	switch t {
	case FmtImage:
		return "image"
	case FmtText:
		fallthrough
	default:
		return "text/plain;charset=utf-8"
	}
}
