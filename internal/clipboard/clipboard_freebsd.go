//go:build freebsd

package clipboard

import (
	"bytes"
	"log/slog"
	"os/exec"
)

func Init() error {
	if _, err := exec.LookPath("xclip"); err == nil {
		return nil
	}
	if _, err := exec.LookPath("xsel"); err == nil {
		return nil
	}
	slog.Warn("neither xclip nor xsel found; clipboard support will be unavailable")
	return nil
}

func Read(t Format) []byte {
	if path, err := exec.LookPath("xclip"); err == nil {
		cmd := exec.Command(path, "-selection", "clipboard", "-o")
		out, err := cmd.Output()
		if err != nil {
			slog.Error("failed to read clipboard via xclip", "err", err)
			return nil
		}
		return out
	}
	if path, err := exec.LookPath("xsel"); err == nil {
		cmd := exec.Command(path, "--clipboard", "--output")
		out, err := cmd.Output()
		if err != nil {
			slog.Error("failed to read clipboard via xsel", "err", err)
			return nil
		}
		return out
	}
	return nil
}

func Write(t Format, buf []byte) <-chan struct{} {
	if path, err := exec.LookPath("xclip"); err == nil {
		cmd := exec.Command(path, "-selection", "clipboard")
		cmd.Stdin = bytes.NewReader(buf)
		if err := cmd.Run(); err != nil {
			slog.Error("failed to write to clipboard via xclip", "err", err)
		}
		return nil
	}
	if path, err := exec.LookPath("xsel"); err == nil {
		cmd := exec.Command(path, "--clipboard", "--input")
		cmd.Stdin = bytes.NewReader(buf)
		if err := cmd.Run(); err != nil {
			slog.Error("failed to write to clipboard via xsel", "err", err)
		}
		return nil
	}
	return nil
}
