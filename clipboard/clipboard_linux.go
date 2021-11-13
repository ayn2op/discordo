//go:build linux

package clipboard

import (
	"errors"
	"os"
	"os/exec"
)

var (
	ErrXclipNotInstalled       = errors.New("xclip is not installed")
	ErrWlClipboardNotInstalled = errors.New("wl-clipboard is not installed")
)

func commandExists(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}

func readXclip() ([]byte, error) {
	if !commandExists("xclip") {
		return nil, ErrXclipNotInstalled
	}

	cmd := exec.Command("xclip", "-selection", "clipboard", "-out")
	return cmd.Output()
}

func writeXclip(in []byte) error {
	if !commandExists("xclip") {
		return ErrXclipNotInstalled
	}

	cmd := exec.Command("xclip", "-selection", "clipboard", "-in")
	wc, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	defer wc.Close()

	err = cmd.Start()
	if err != nil {
		return err
	}

	_, err = wc.Write(in)
	if err != nil {
		return err
	}

	return nil
}

func readWlClipboard() ([]byte, error) {
	if !commandExists("wl-paste") {
		return nil, ErrWlClipboardNotInstalled
	}

	cmd := exec.Command("wl-paste", "--no-newline")
	return cmd.Output()
}

func writeWlClipboard(in []byte) error {
	if !commandExists("wl-copy") {
		return ErrWlClipboardNotInstalled
	}

	cmd := exec.Command("wl-copy")
	wc, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	defer wc.Close()

	err = cmd.Start()
	if err != nil {
		return err
	}

	_, err = wc.Write(in)
	if err != nil {
		return err
	}

	return nil
}

func read() ([]byte, error) {
	if os.Getenv("XDG_SESSION_TYPE") == "wayland" {
		return readWlClipboard()
	} else {
		return readXclip()
	}
}

func write(in []byte) error {
	if os.Getenv("XDG_SESSION_TYPE") == "wayland" {
		return writeWlClipboard(in)
	} else {
		return writeXclip(in)
	}
}
