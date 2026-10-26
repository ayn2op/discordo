//go:build !no_voice

package voice

import (
	"bytes"
	"io"
	"testing"
	"time"

	"github.com/tphakala/go-opus/oggopus"
)

func TestPlayerSeek(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		const seconds = 10

		var audio bytes.Buffer
		if err := oggopus.EncodeInterleaved(&audio, oggopus.Config{SampleRate: sampleRate, Channels: 1}, make([]byte, seconds*sampleRate*sampleBytes)); err != nil {
			t.Fatal(err)
		}

		p := Player{audio: audio.Bytes()}
		for _, position := range []time.Duration{0, 500 * time.Millisecond, 6 * time.Second, seconds * time.Second, time.Hour} {
			t.Run(position.String(), func(t *testing.T) {
				if err := p.seek(position); err != nil {
					t.Fatal(err)
				}

				left, err := io.Copy(io.Discard, p.decoder)
				if err != nil {
					t.Fatal(err)
				}

				// What is left is a packet longer at most, as the end is only trimmed when played from the start.
				want := max(seconds*time.Second-position, 0)
				if got := time.Duration(left/2) * time.Second / sampleRate; got < want || got > want+20*time.Millisecond {
					t.Errorf("%v is left, want %v", got, want)
				}
			})
		}
	})

	// A page whose segment table claims far more bytes than the file holds must not make seek read out of bounds.
	t.Run("truncated page", func(t *testing.T) {
		page := make([]byte, oggHeaderSize+255)
		copy(page, "OggS")
		page[oggSegmentsOffset] = 255
		for i := range page[oggHeaderSize:] {
			page[oggHeaderSize+i] = 255
		}

		p := Player{audio: page}
		_ = p.seek(0)
	})
}
