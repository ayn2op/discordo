package voice

import (
	"gopkg.in/hraban/opus.v2"
)

const (
	opusSampleRate = 48000
	opusFrameSize  = 960 // 20ms at 48kHz
	opusBitrate    = 64000
	encodeChannels = 1
	decodeChannels = 2
	maxOpusPacket  = 4000
)

// OpusEncoder wraps an opus encoder for mono voice capture.
type OpusEncoder struct {
	enc *opus.Encoder
}

// NewOpusEncoder creates a new Opus encoder configured for voice at 48kHz mono.
func NewOpusEncoder() (*OpusEncoder, error) {
	enc, err := opus.NewEncoder(opusSampleRate, encodeChannels, opus.AppVoIP)
	if err != nil {
		return nil, err
	}

	if err := enc.SetBitrate(opusBitrate); err != nil {
		return nil, err
	}

	return &OpusEncoder{enc: enc}, nil
}

// Encode encodes a mono PCM frame into Opus data.
func (e *OpusEncoder) Encode(pcm []int16) ([]byte, error) {
	buf := make([]byte, maxOpusPacket)
	n, err := e.enc.Encode(pcm, buf)
	if err != nil {
		return nil, err
	}
	return buf[:n], nil
}

// OpusDecoder wraps an opus decoder for stereo playback.
type OpusDecoder struct {
	dec *opus.Decoder
}

// NewOpusDecoder creates a new Opus decoder configured for 48kHz stereo output.
func NewOpusDecoder() (*OpusDecoder, error) {
	dec, err := opus.NewDecoder(opusSampleRate, decodeChannels)
	if err != nil {
		return nil, err
	}
	return &OpusDecoder{dec: dec}, nil
}

// Decode decodes Opus data into a stereo PCM frame.
func (d *OpusDecoder) Decode(opusData []byte) ([]int16, error) {
	pcm := make([]int16, opusFrameSize*decodeChannels)
	n, err := d.dec.Decode(opusData, pcm)
	if err != nil {
		return nil, err
	}
	return pcm[:n*decodeChannels], nil
}

// Close is a no-op provided for interface consistency.
func (d *OpusDecoder) Close() {}
