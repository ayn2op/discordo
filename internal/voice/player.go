//go:build !no_voice

package voice

import (
	"bytes"
	"encoding/binary"
	"io"
	"sync"
	"time"

	"github.com/ayn2op/arikawa/v3/discord"
	"github.com/gen2brain/malgo"
	"github.com/tphakala/go-opus/oggopus"
)

const (
	// The sizes of the header of an Ogg page, and where in it its flags, its granule position and how many segments it has are.
	oggHeaderSize     = 27
	oggFlagsOffset    = 5
	oggGranuleOffset  = 6
	oggSegmentsOffset = 26
	// oggContinued is the flag of a page that begins with the rest of a packet.
	oggContinued = 1

	// progressSamples is how many samples are played between updates.
	progressSamples = sampleRate / 10
)

// Player plays voice messages, one at a time.
type Player struct {
	updates chan struct{}

	mu sync.Mutex
	// device plays the voice message that is the attachment id, and is nil once it ended.
	device *malgo.Device
	id     discord.AttachmentID
	// audio is the Ogg Opus file, decoder what is left of it to play, and samples how much of it was played.
	audio   []byte
	decoder *oggopus.Decoder
	samples int64
}

func NewPlayer() *Player {
	return &Player{updates: make(chan struct{}, 1)}
}

// Updates is sent to when the status changes.
func (p *Player) Updates() <-chan struct{} {
	return p.updates
}

func (p *Player) update() {
	select {
	case p.updates <- struct{}{}:
	default:
	}
}

func (p *Player) Status() PlayerStatus {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.device == nil {
		return PlayerStatus{}
	}

	return PlayerStatus{ID: p.id, Position: time.Duration(p.samples) * time.Second / sampleRate, Playing: p.device.IsStarted()}
}

// Toggle pauses or resumes the voice message that is the attachment id, and reports whether it is the one that Play was last given and has not ended.
func (p *Player) Toggle(id discord.AttachmentID) (bool, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	defer p.update()

	switch {
	case p.device == nil || p.id != id:
		return false, nil
	case p.device.IsStarted():
		return true, p.device.Stop()
	default:
		return true, p.device.Start()
	}
}

// stop stops device if it is what is playing.
func (p *Player) stop(device *malgo.Device) {
	if device != nil && p.device == device {
		device.Uninit()
		p.device = nil
		p.update()
	}
}

// seek makes what is left to play start at position, by decoding from the last page that ends before it, as the decoder cannot seek.
func (p *Player) seek(position time.Duration) error {
	sample := int64(max(position, 0) * sampleRate / time.Second)

	// headers is where the pages before the audio end, start the page to decode from, and first the sample it starts with.
	var headers, start int
	var first, last int64
	for offset := 0; offset+oggHeaderSize <= len(p.audio); {
		page := p.audio[offset:]
		segments := page[oggHeaderSize:][:min(int(page[oggSegmentsOffset]), len(page)-oggHeaderSize)]
		size := oggHeaderSize + len(segments)
		for _, segment := range segments {
			size += int(segment)
		}

		granule := int64(binary.LittleEndian.Uint64(page[oggGranuleOffset:]))
		switch {
		case granule == 0:
			headers, start = offset+size, offset+size
		case last <= sample && page[oggFlagsOffset]&oggContinued == 0:
			start, first = offset, last
		}

		// A page that no packet ends on has no position.
		if granule > 0 {
			last = granule
		}

		offset += size
	}

	decoder, err := oggopus.NewDecoder(io.MultiReader(bytes.NewReader(p.audio[:headers]), bytes.NewReader(p.audio[min(start, len(p.audio)):])))
	if err != nil {
		return err
	}

	// Nothing is left to play if position is past the end.
	io.CopyN(io.Discard, decoder, (sample-first)*int64(decoder.Info().Channels)*2)
	p.decoder, p.samples = decoder, sample

	return nil
}

// Seek moves the voice message that is playing or paused to position.
func (p *Player) Seek(position time.Duration) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	defer p.update()

	if p.device == nil {
		return nil
	}

	return p.seek(position)
}

// Play plays the voice message that is the attachment id, which is the Ogg Opus file audio, in place of the one that is playing.
func (p *Player) Play(id discord.AttachmentID, audio []byte) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.stop(p.device)

	p.audio = audio
	if err := p.seek(0); err != nil {
		return err
	}

	var device *malgo.Device
	ended := sync.OnceFunc(func() {
		p.mu.Lock()
		defer p.mu.Unlock()

		p.stop(device)
	})

	channels := p.decoder.Info().Channels
	device, err := startDevice(malgo.Playback, channels, func(output, _ []byte) {
		// What is played is left silent rather than wait for Seek, and for Play, which has not set the device yet.
		if !p.mu.TryLock() {
			return
		}
		defer p.mu.Unlock()

		if p.device != device {
			return
		}

		n, err := io.ReadFull(p.decoder, output)
		played := p.samples + int64(n/channels/2)
		if played/progressSamples != p.samples/progressSamples {
			p.update()
		}

		p.samples = played

		// A device cannot be stopped by what it calls.
		if err != nil {
			go ended()
		}
	})
	p.device, p.id = device, id
	p.update()

	return err
}
