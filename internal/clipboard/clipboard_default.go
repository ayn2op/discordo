//go:build !linux && !freebsd

package clipboard

import (
	"github.com/ayn2op/clipboard"
)

func Init() error {
	return clipboard.Init()
}

func Read(t Format) ([]byte, error) {
	return clipboard.Read(clipboard.Format(t))
}

func Write(t Format, buf []byte) error {
	return clipboard.Write(clipboard.Format(t), buf)
}
