//go:build !linux

package clipboard

import designClipb "golang.design/x/clipboard"

var (
	Init = designClipb.Init
	Read = designClipb.Read
	Write = designClipb.Write
)
