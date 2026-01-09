package media

import (
	"errors"
	"os"
	"strings"
	"sync"

	"github.com/BourgeoisBear/rasterm"
)

type Protocol int

const (
	ProtoAuto Protocol = iota
	ProtoKitty
	ProtoIterm
	ProtoSixel
	ProtoAnsi
	ProtoFallback
)

var protocolNames = map[Protocol]string{
	ProtoKitty:    "Kitty",
	ProtoIterm:    "iTerm2",
	ProtoSixel:    "Sixel",
	ProtoAnsi:     "ANSI",
	ProtoFallback: "Fallback",
	ProtoAuto:     "Auto",
}

func (p Protocol) String() string {
	if name, ok := protocolNames[p]; ok {
		return name
	}
	return "Unknown"
}

var ErrInvalidProtocol = errors.New("invalid image protocol")

var protocolConfigMap = map[string]Protocol{
	"":      ProtoAuto,
	"auto":  ProtoAuto,
	"kitty": ProtoKitty,
	"sixel": ProtoSixel,
	"iterm": ProtoIterm,
	"ascii": ProtoAnsi,
	"ansi":  ProtoAnsi,
}

func ParseProtocol(s string) (Protocol, error) {
	if proto, ok := protocolConfigMap[strings.ToLower(s)]; ok {
		return proto, nil
	}
	return ProtoAuto, ErrInvalidProtocol
}

var (
	protocolMu      sync.RWMutex
	currentProtocol Protocol
	protocolOnce    sync.Once
	forcedProtocol  *Protocol
)

func SetProtocol(p Protocol) {
	protocolMu.Lock()
	defer protocolMu.Unlock()
	forcedProtocol = &p
}

var multiplexerEnvVars = []string{
	"TMUX",
	"ZELLIJ",
	"STY",
}

func isMultiplexer() bool {
	for _, env := range multiplexerEnvVars {
		if os.Getenv(env) != "" {
			return true
		}
	}
	return false
}

type protocolDetector func() (Protocol, bool)

var protocolDetectors = []protocolDetector{
	func() (Protocol, bool) {
		if isMultiplexer() {
			return ProtoAnsi, true
		}
		return ProtoAuto, false
	},
	func() (Protocol, bool) {
		if rasterm.IsKittyCapable() {
			return ProtoKitty, true
		}
		return ProtoAuto, false
	},
	func() (Protocol, bool) {
		if rasterm.IsItermCapable() {
			return ProtoIterm, true
		}
		return ProtoAuto, false
	},
	func() (Protocol, bool) {
		term := os.Getenv("TERM")
		if term == "foot" || term == "foot-extra" {
			return ProtoSixel, true
		}
		return ProtoAuto, false
	},
	func() (Protocol, bool) {
		if capable, err := rasterm.IsSixelCapable(); err == nil && capable {
			return ProtoSixel, true
		}
		return ProtoAuto, false
	},
}

func DetectProtocol() Protocol {
	protocolMu.RLock()
	if forcedProtocol != nil {
		p := *forcedProtocol
		protocolMu.RUnlock()
		return p
	}
	protocolMu.RUnlock()

	protocolOnce.Do(func() {
		for _, detect := range protocolDetectors {
			if proto, ok := detect(); ok {
				currentProtocol = proto
				return
			}
		}
		currentProtocol = ProtoAnsi
	})
	return currentProtocol
}
