package media

import (
	"errors"
	"sync"
	"testing"
)

func resetProtocolState() {
	protocolMu.Lock()
	defer protocolMu.Unlock()
	forcedProtocol = nil
	protocolOnce = sync.Once{}
	currentProtocol = ProtoAuto
}

func TestParseProtocol(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		want      Protocol
		wantError bool
	}{
		{
			name:      "empty string returns ProtoAuto",
			input:     "",
			want:      ProtoAuto,
			wantError: false,
		},
		{
			name:      "auto lowercase returns ProtoAuto",
			input:     "auto",
			want:      ProtoAuto,
			wantError: false,
		},
		{
			name:      "AUTO uppercase returns ProtoAuto",
			input:     "AUTO",
			want:      ProtoAuto,
			wantError: false,
		},
		{
			name:      "kitty lowercase",
			input:     "kitty",
			want:      ProtoKitty,
			wantError: false,
		},
		{
			name:      "KITTY uppercase",
			input:     "KITTY",
			want:      ProtoKitty,
			wantError: false,
		},
		{
			name:      "Kitty mixed case",
			input:     "Kitty",
			want:      ProtoKitty,
			wantError: false,
		},
		{
			name:      "sixel lowercase",
			input:     "sixel",
			want:      ProtoSixel,
			wantError: false,
		},
		{
			name:      "Sixel mixed case",
			input:     "Sixel",
			want:      ProtoSixel,
			wantError: false,
		},
		{
			name:      "SIXEL uppercase",
			input:     "SIXEL",
			want:      ProtoSixel,
			wantError: false,
		},
		{
			name:      "iterm lowercase",
			input:     "iterm",
			want:      ProtoIterm,
			wantError: false,
		},
		{
			name:      "ITerm mixed case",
			input:     "ITerm",
			want:      ProtoIterm,
			wantError: false,
		},
		{
			name:      "ascii lowercase",
			input:     "ascii",
			want:      ProtoAnsi,
			wantError: false,
		},
		{
			name:      "ASCII uppercase",
			input:     "ASCII",
			want:      ProtoAnsi,
			wantError: false,
		},
		{
			name:      "ansi lowercase",
			input:     "ansi",
			want:      ProtoAnsi,
			wantError: false,
		},
		{
			name:      "ANSI uppercase",
			input:     "ANSI",
			want:      ProtoAnsi,
			wantError: false,
		},
		{
			name:      "invalid protocol",
			input:     "invalid",
			want:      ProtoAuto,
			wantError: true,
		},
		{
			name:      "unknown protocol",
			input:     "unknown",
			want:      ProtoAuto,
			wantError: true,
		},
		{
			name:      "numeric value",
			input:     "123",
			want:      ProtoAuto,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseProtocol(tt.input)

			if tt.wantError {
				if err == nil {
					t.Errorf("ParseProtocol(%q) expected error, got nil", tt.input)
				}
				if !errors.Is(err, ErrInvalidProtocol) {
					t.Errorf("ParseProtocol(%q) error = %v, want ErrInvalidProtocol", tt.input, err)
				}
			} else {
				if err != nil {
					t.Errorf("ParseProtocol(%q) unexpected error: %v", tt.input, err)
				}
			}

			if got != tt.want {
				t.Errorf("ParseProtocol(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestProtocolString(t *testing.T) {
	tests := []struct {
		proto Protocol
		want  string
	}{
		{ProtoAuto, "Auto"},
		{ProtoKitty, "Kitty"},
		{ProtoIterm, "iTerm2"},
		{ProtoSixel, "Sixel"},
		{ProtoAnsi, "ANSI"},
		{ProtoFallback, "Fallback"},
		{Protocol(99), "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.proto.String(); got != tt.want {
				t.Errorf("Protocol(%d).String() = %v, want %v", tt.proto, got, tt.want)
			}
		})
	}
}

func TestIsMultiplexer(t *testing.T) {
	tests := []struct {
		name    string
		envVars map[string]string
		want    bool
	}{
		{
			name:    "no multiplexer env vars",
			envVars: nil,
			want:    false,
		},
		{
			name:    "TMUX set",
			envVars: map[string]string{"TMUX": "/tmp/tmux-1000/default,12345,0"},
			want:    true,
		},
		{
			name:    "ZELLIJ set",
			envVars: map[string]string{"ZELLIJ": "0"},
			want:    true,
		},
		{
			name:    "STY set (screen)",
			envVars: map[string]string{"STY": "12345.pts-0.hostname"},
			want:    true,
		},
		{
			name:    "multiple multiplexers",
			envVars: map[string]string{"TMUX": "value", "ZELLIJ": "value"},
			want:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, env := range multiplexerEnvVars {
				t.Setenv(env, "")
			}
			for k, v := range tt.envVars {
				t.Setenv(k, v)
			}

			if got := isMultiplexer(); got != tt.want {
				t.Errorf("isMultiplexer() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSetProtocol(t *testing.T) {
	t.Cleanup(resetProtocolState)

	tests := []struct {
		name     string
		protocol Protocol
	}{
		{"set Kitty", ProtoKitty},
		{"set Sixel", ProtoSixel},
		{"set iTerm", ProtoIterm},
		{"set ANSI", ProtoAnsi},
		{"set Fallback", ProtoFallback},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetProtocolState()
			SetProtocol(tt.protocol)

			got := DetectProtocol()
			if got != tt.protocol {
				t.Errorf("after SetProtocol(%v), DetectProtocol() = %v, want %v", tt.protocol, got, tt.protocol)
			}
		})
	}
}

func TestDetectProtocolWithForcedProtocol(t *testing.T) {
	t.Cleanup(resetProtocolState)

	resetProtocolState()
	SetProtocol(ProtoSixel)

	got := DetectProtocol()
	if got != ProtoSixel {
		t.Errorf("DetectProtocol() with forced = %v, want %v", got, ProtoSixel)
	}

	SetProtocol(ProtoKitty)
	got = DetectProtocol()
	if got != ProtoKitty {
		t.Errorf("DetectProtocol() after changing forced = %v, want %v", got, ProtoKitty)
	}
}

func TestDetectProtocolMultiplexerFallback(t *testing.T) {
	t.Cleanup(resetProtocolState)

	for _, env := range multiplexerEnvVars {
		t.Setenv(env, "")
	}
	t.Setenv("TMUX", "/tmp/tmux-1000/default,12345,0")

	resetProtocolState()

	got := DetectProtocol()
	if got != ProtoAnsi {
		t.Errorf("DetectProtocol() in multiplexer = %v, want %v", got, ProtoAnsi)
	}
}
