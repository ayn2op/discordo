package app

import (
	"errors"
	"testing"

	"github.com/ayn2op/discordo/internal/media"
)

func TestParseProtocolConfig(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		want      media.Protocol
		wantError bool
	}{
		{
			name:      "empty string returns ProtoAuto",
			input:     "",
			want:      media.ProtoAuto,
			wantError: false,
		},
		{
			name:      "kitty lowercase",
			input:     "kitty",
			want:      media.ProtoKitty,
			wantError: false,
		},
		{
			name:      "KITTY uppercase",
			input:     "KITTY",
			want:      media.ProtoKitty,
			wantError: false,
		},
		{
			name:      "Kitty mixed case",
			input:     "Kitty",
			want:      media.ProtoKitty,
			wantError: false,
		},
		{
			name:      "sixel lowercase",
			input:     "sixel",
			want:      media.ProtoSixel,
			wantError: false,
		},
		{
			name:      "Sixel mixed case",
			input:     "Sixel",
			want:      media.ProtoSixel,
			wantError: false,
		},
		{
			name:      "SIXEL uppercase",
			input:     "SIXEL",
			want:      media.ProtoSixel,
			wantError: false,
		},
		{
			name:      "iterm lowercase",
			input:     "iterm",
			want:      media.ProtoIterm,
			wantError: false,
		},
		{
			name:      "ITerm mixed case",
			input:     "ITerm",
			want:      media.ProtoIterm,
			wantError: false,
		},
		{
			name:      "ascii lowercase",
			input:     "ascii",
			want:      media.ProtoAnsi,
			wantError: false,
		},
		{
			name:      "ASCII uppercase",
			input:     "ASCII",
			want:      media.ProtoAnsi,
			wantError: false,
		},
		{
			name:      "ansi lowercase",
			input:     "ansi",
			want:      media.ProtoAnsi,
			wantError: false,
		},
		{
			name:      "ANSI uppercase",
			input:     "ANSI",
			want:      media.ProtoAnsi,
			wantError: false,
		},
		{
			name:      "invalid protocol",
			input:     "invalid",
			want:      media.ProtoAuto,
			wantError: true,
		},
		{
			name:      "unknown protocol",
			input:     "unknown",
			want:      media.ProtoAuto,
			wantError: true,
		},
		{
			name:      "numeric value",
			input:     "123",
			want:      media.ProtoAuto,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseProtocolConfig(tt.input)

			if tt.wantError {
				if err == nil {
					t.Errorf("parseProtocolConfig(%q) expected error, got nil", tt.input)
				}
				if !errors.Is(err, ErrInvalidProtocol) {
					t.Errorf("parseProtocolConfig(%q) error = %v, want ErrInvalidProtocol", tt.input, err)
				}
			} else {
				if err != nil {
					t.Errorf("parseProtocolConfig(%q) unexpected error: %v", tt.input, err)
				}
			}

			if got != tt.want {
				t.Errorf("parseProtocolConfig(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}
