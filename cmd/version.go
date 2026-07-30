package cmd

import (
	"runtime/debug"
)

func buildVersion() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "unknown"
	}
	switch info.Main.Version {
	case "", "(devel)":
		return "v0.0.0-dev"
	default:
		return info.Main.Version
	}
}
