package voice

import (
	"log/slog"
	"math"
	"sync"
	"time"
)

const (
	jitterBufSize = 3
	staleTimeout  = 200 * time.Millisecond
)

type decoderState struct {
	decoder  *OpusDecoder
	buffer   [][]int16 // ring buffer for jitter smoothing
	writeIdx int
	readIdx  int
	count    int
	lastSeen time.Time
}

// Mixer handles mixing audio from multiple users identified by SSRC.
type Mixer struct {
	mu       sync.Mutex
	decoders map[uint32]*decoderState
}

// NewMixer creates a new audio mixer.
func NewMixer() *Mixer {
	return &Mixer{
		decoders: make(map[uint32]*decoderState),
	}
}

// AddPacket decodes an Opus packet for the given SSRC and buffers it.
func (m *Mixer) AddPacket(ssrc uint32, opusData []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	ds, ok := m.decoders[ssrc]
	if !ok {
		dec, err := NewOpusDecoder()
		if err != nil {
			return err
		}
		ds = &decoderState{
			decoder: dec,
			buffer:  make([][]int16, jitterBufSize),
		}
		m.decoders[ssrc] = ds
	}

	ds.lastSeen = time.Now()

	pcm, err := ds.decoder.Decode(opusData)
	if err != nil {
		slog.Error("failed to decode opus packet", "ssrc", ssrc, "err", err)
		return err
	}

	ds.buffer[ds.writeIdx] = pcm
	ds.writeIdx = (ds.writeIdx + 1) % jitterBufSize
	if ds.count < jitterBufSize {
		ds.count++
	} else {
		// overwrite oldest unread
		ds.readIdx = (ds.readIdx + 1) % jitterBufSize
	}

	return nil
}

// Mix returns the mixed stereo PCM output from all active sources.
func (m *Mixer) Mix() []int16 {
	m.mu.Lock()
	defer m.mu.Unlock()

	mixed := make([]int16, opusFrameSize*decodeChannels)
	for _, ds := range m.decoders {
		if ds.count == 0 {
			continue
		}

		frame := ds.buffer[ds.readIdx]
		ds.readIdx = (ds.readIdx + 1) % jitterBufSize
		ds.count--

		if frame == nil {
			continue
		}

		for i := range min(len(mixed), len(frame)) {
			sum := int32(mixed[i]) + int32(frame[i])
			mixed[i] = clampInt16(sum)
		}
	}

	return mixed
}

// Cleanup removes decoders that haven't received data recently.
func (m *Mixer) Cleanup() {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	for ssrc, ds := range m.decoders {
		if now.Sub(ds.lastSeen) > staleTimeout {
			ds.decoder.Close()
			delete(m.decoders, ssrc)
		}
	}
}

// Close releases all decoder resources.
func (m *Mixer) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for ssrc, ds := range m.decoders {
		ds.decoder.Close()
		delete(m.decoders, ssrc)
	}
}

func clampInt16(v int32) int16 {
	if v > math.MaxInt16 {
		return math.MaxInt16
	}
	if v < math.MinInt16 {
		return math.MinInt16
	}
	return int16(v)
}
