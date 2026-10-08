package voice

import (
	"encoding/binary"
	"sync"
	"time"

	"github.com/tphakala/go-opus/opus"
)

const (
	sampleRate = 48000
	channels   = 2
	// frameSamples and frameBytes are the sizes of 20ms of PCM.
	frameSamples = sampleRate / 50 * channels
	frameBytes   = frameSamples * 2

	// cushion is how much is buffered before it is played, so that packets arriving unevenly do not make gaps.
	cushion = 3 * frameBytes
	// maxBuffered is how much is buffered before the oldest is dropped rather than fall behind.
	maxBuffered = 10 * frameBytes
	// maxLost is how many packets in a row are made up for.
	maxLost = 3
)

// speaker decodes and buffers what one user says, to be mixed into what is played.
type speaker struct {
	mu      sync.Mutex
	decoder *opus.Decoder
	pcm     []int16
	buf     []byte
	// playing is whether the cushion was buffered and what is left of it is being played.
	playing bool
	// seq is the sequence number of the last packet, and last when it arrived.
	seq  uint16
	last time.Time
}

// newSpeaker returns a speaker whose first packet has the sequence number seq.
func newSpeaker(seq uint16) (*speaker, error) {
	decoder, err := opus.NewDecoder(sampleRate, channels)
	if err != nil {
		return nil, err
	}
	// 120ms is the longest an Opus packet can be.
	return &speaker{decoder: decoder, pcm: make([]int16, 6*frameSamples), seq: seq - 1}, nil
}

// decode appends what packet decodes to, or a made up frame if it is nil.
func (s *speaker) decode(packet []byte) error {
	pcm := s.pcm
	if packet == nil {
		pcm = pcm[:frameSamples]
	}
	n, err := s.decoder.Decode(packet, pcm)
	if err != nil {
		return err
	}
	for _, sample := range pcm[:n*channels] {
		s.buf = binary.LittleEndian.AppendUint16(s.buf, uint16(sample))
	}
	return nil
}

// write buffers the packet with the sequence number seq.
func (s *speaker) write(seq uint16, packet []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Make up for the few packets that never arrived. One that arrives late or twice is not worth playing.
	gap := seq - s.seq
	if gap == 0 || gap > 1<<15 && time.Since(s.last) < time.Second {
		return nil
	}
	if gap > 1 && gap <= maxLost+1 {
		for range gap - 1 {
			s.decode(nil)
		}
	}
	s.seq, s.last = seq, time.Now()

	if err := s.decode(packet); err != nil {
		return err
	}
	if len(s.buf) > maxBuffered {
		s.buf = s.buf[len(s.buf)-cushion:]
	}
	return nil
}

// mix adds what is buffered to output, once the cushion is.
func (s *speaker) mix(output []byte) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.playing = s.playing || len(s.buf) >= cushion; !s.playing {
		return
	}
	n := min(len(output), len(s.buf))
	for i := 0; i+1 < n; i += 2 {
		sum := int32(int16(binary.LittleEndian.Uint16(output[i:]))) + int32(int16(binary.LittleEndian.Uint16(s.buf[i:])))
		binary.LittleEndian.PutUint16(output[i:], uint16(max(-1<<15, min(1<<15-1, sum))))
	}
	s.buf = s.buf[n:]
	s.playing = len(s.buf) > 0
}
