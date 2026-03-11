//go:build linux || freebsd

package clipboard

import (
	"bytes"
	"github.com/ayn2op/clipboard"
	"os"
	"os/exec"
)

var wayland bool

func Init() error {
	if _, ok := os.LookupEnv("WAYLAND_DISPLAY"); !ok {
		return clipboard.Init()
	}
	if _, err := exec.LookPath("wl-copy"); err != nil {
		return err
	}
	if _, err := exec.LookPath("wl-paste"); err != nil {
		return err
	}
	wayland = true
	return nil
}

func Read(t Format) ([]byte, error) {
	if !wayland {
		return clipboard.Read(clipboard.Format(t))
	}
	// -n: Don't print a newline at the end
	// -t type: MIME type specifier
	cmd := exec.Command("wl-paste", "-nt", formatToType(t))
	outBuffer := bytes.Buffer{}
	cmd.Stdout = &outBuffer
	if err := cmd.Run(); err != nil {
		return nil, err
	}
	return outBuffer.Bytes(), nil
}

func Write(t Format, buf []byte) error {
	if !wayland {
		return clipboard.Write(clipboard.Format(t), buf)
	}
	// -t type: MIME type specifier
	cmd := exec.Command("wl-copy", "-t", formatToType(t))
	cmd.Stdin = bytes.NewReader(buf)
	return cmd.Run()
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
