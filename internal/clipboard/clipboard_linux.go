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

func Init() error {
	if wayland = checkWaylandEnv(); !wayland {
		return designClipb.Init()
	}
	return nil
}

func checkWaylandEnv() bool {
	if _, ok := os.LookupEnv("WAYLAND_DISPLAY"); !ok {
		return false
	}
	if _, err := exec.LookPath("wl-copy"); err != nil {
		return false
	}
	_, err := exec.LookPath("wl-paste")
	return err == nil
}

func Read(t Format) []byte {
	if !wayland {
		return designClipb.Read(designClipb.Format(t))
	}
	var typ string
	switch t {
	case FmtText:
		typ = "text/plain;charset=utf-8"
	case FmtImage:
		typ = "image"
	}
	cmd := exec.Command("wl-paste", "--no-newline", "--type", typ)
	outBuffer := bytes.Buffer{}
	cmd.Stdout = &outBuffer
	if err := cmd.Run(); err != nil {
		slog.Error("failed to write to clipboard", "err", err)
		return nil
	}
	return outBuffer.Bytes()
}

func Write(t Format, buf []byte) <-chan struct{} {
	if !wayland {
		return designClipb.Write(designClipb.Format(t), buf)
	}
	var typ string
	switch t {
	case FmtText:
		typ = "text/plain;charset=utf-8"
	case FmtImage:
		typ = "image"
	}
	cmd := exec.Command("wl-copy", "--type", typ)
	cmd.Stdin = bytes.NewReader(buf)
	if err := cmd.Run(); err != nil {
		slog.Error("failed to write to clipboard", "err", err)
	}
	return nil
}
