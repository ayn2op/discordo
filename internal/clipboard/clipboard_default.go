//go:build !linux

package clipboard

import designClipb "github.com/ayn2op/clipboard"

var (
	Init  = designClipb.Init
	Read  = designClipb.Read
	Write = designClipb.Write
)
