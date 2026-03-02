package clipboard

import (
	"log/slog"

	nativeclipboard "github.com/aymanbagabas/go-nativeclipboard"
)

// Format represents the type of clipboard content.
type Format int

const (
	FmtText  Format = iota // plain text
	FmtImage               // image data
)

func Init() error {
	return nil
}

func Read(t Format) []byte {
	f := formatToNative(t)
	data, err := f.Read()
	if err != nil {
		slog.Error("failed to read clipboard", "err", err)
		return nil
	}
	return data
}

func Write(t Format, buf []byte) <-chan struct{} {
	f := formatToNative(t)
	if _, err := f.Write(buf); err != nil {
		slog.Error("failed to write to clipboard", "err", err)
	}
	return nil
}

func formatToNative(t Format) nativeclipboard.Format {
	switch t {
	case FmtImage:
		return nativeclipboard.Image
	default:
		return nativeclipboard.Text
	}
}
