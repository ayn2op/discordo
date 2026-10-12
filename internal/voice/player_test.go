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
	const seconds = 10

	var audio bytes.Buffer
	if err := oggopus.EncodeInterleaved(&audio, oggopus.Config{SampleRate: sampleRate, Channels: 1}, make([]byte, seconds*sampleRate*2)); err != nil {
		t.Fatal(err)
	}

	p := Player{audio: audio.Bytes()}
	for _, position := range []time.Duration{0, 500 * time.Millisecond, 6 * time.Second, seconds * time.Second, time.Hour} {
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
			t.Errorf("seek to %v: %v is left, want %v", position, got, want)
		}
	}
}
