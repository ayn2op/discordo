//go:build !linux

package clipboard

import designClipb "golang.design/x/clipboard"

func Init() error {
	return designClipb.Init()
}

func Read(t Format) []byte {
	return designClipb.Read(designClipb.Format(t))
}

func Write(t Format, buf []byte) <-chan struct{} {
	return designClipb.Write(designClipb.Format(t), buf)
}
