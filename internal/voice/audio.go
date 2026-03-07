package voice

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/gordonklaus/portaudio"
)

const (
	sampleRate  = 48000
	channels    = 1
	frameSize   = 960 // 20ms at 48kHz
	stereo      = 2
	stereoFrame = frameSize * stereo
)

// InitAudio initializes the PortAudio library.
func InitAudio() error {
	return portaudio.Initialize()
}

// TerminateAudio shuts down the PortAudio library.
func TerminateAudio() error {
	return portaudio.Terminate()
}

// AudioDevice manages audio capture and playback streams.
type AudioDevice struct {
	mu         sync.Mutex
	inputName  string
	outputName string
	inputVol   float64
	outputVol  float64

	captureStream  *portaudio.Stream
	playbackStream *portaudio.Stream
}

// NewAudioDevice creates a new audio device with the given volume levels.
func NewAudioDevice(inputName, outputName string, inputVol, outputVol float64) *AudioDevice {
	return &AudioDevice{
		inputName:  inputName,
		outputName: outputName,
		inputVol:   inputVol,
		outputVol:  outputVol,
	}
}

// StartCapture opens the microphone and returns a channel of PCM frames.
// Each frame contains 960 mono int16 samples (20ms at 48kHz).
func (d *AudioDevice) StartCapture(ctx context.Context) (<-chan []int16, error) {
	out := make(chan []int16, 8)
	buf := make([]int16, frameSize)

	inDev, err := resolveInputDevice(d.inputName)
	if err != nil {
		close(out)
		return nil, err
	}

	params := portaudio.LowLatencyParameters(inDev, nil)
	params.SampleRate = sampleRate
	params.FramesPerBuffer = frameSize

	stream, err := portaudio.OpenStream(params, buf)
	if err != nil {
		close(out)
		return nil, err
	}

	if err := stream.Start(); err != nil {
		stream.Close()
		close(out)
		return nil, err
	}

	d.mu.Lock()
	d.captureStream = stream
	d.mu.Unlock()

	go func() {
		defer close(out)
		defer func() {
			if err := stream.Stop(); err != nil {
				slog.Error("failed to stop capture stream", "err", err)
			}
			if err := stream.Close(); err != nil {
				slog.Error("failed to close capture stream", "err", err)
			}
		}()

		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			if err := stream.Read(); err != nil {
				// Input overflow is normal during brief processing delays.
				if err != portaudio.InputOverflowed {
					slog.Error("failed to read from capture stream", "err", err)
					return
				}
			}

			d.mu.Lock()
			vol := d.inputVol
			d.mu.Unlock()

			frame := make([]int16, frameSize)
			for i, s := range buf {
				frame[i] = int16(float64(s) * vol)
			}

			select {
			case out <- frame:
			case <-ctx.Done():
				return
			}
		}
	}()

	return out, nil
}

// StartPlayback opens the speaker and returns a channel to send PCM frames.
// Each frame should contain 960 stereo int16 samples (1920 samples total, 20ms at 48kHz).
func (d *AudioDevice) StartPlayback(ctx context.Context) (chan<- []int16, error) {
	in := make(chan []int16, 8)
	buf := make([]int16, stereoFrame)

	outDev, err := resolveOutputDevice(d.outputName)
	if err != nil {
		close(in)
		return nil, err
	}

	params := portaudio.LowLatencyParameters(nil, outDev)
	params.SampleRate = sampleRate
	params.FramesPerBuffer = frameSize

	stream, err := portaudio.OpenStream(params, buf)
	if err != nil {
		close(in)
		return nil, err
	}

	if err := stream.Start(); err != nil {
		stream.Close()
		close(in)
		return nil, err
	}

	d.mu.Lock()
	d.playbackStream = stream
	d.mu.Unlock()

	go func() {
		defer func() {
			if err := stream.Stop(); err != nil {
				slog.Error("failed to stop playback stream", "err", err)
			}
			if err := stream.Close(); err != nil {
				slog.Error("failed to close playback stream", "err", err)
			}
		}()

		for {
			select {
			case frame, ok := <-in:
				if !ok {
					return
				}

				d.mu.Lock()
				vol := d.outputVol
				d.mu.Unlock()

				for i := range min(len(frame), len(buf)) {
					buf[i] = int16(float64(frame[i]) * vol)
				}

				if err := stream.Write(); err != nil {
					// Output underflow is normal when there are gaps in
					// incoming audio packets — just continue.
					if err != portaudio.OutputUnderflowed {
						slog.Error("failed to write to playback stream", "err", err)
						return
					}
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	return in, nil
}

// SetInputVolume sets the microphone volume multiplier.
func (d *AudioDevice) SetInputVolume(vol float64) {
	d.mu.Lock()
	d.inputVol = vol
	d.mu.Unlock()
}

// SetOutputVolume sets the speaker volume multiplier.
func (d *AudioDevice) SetOutputVolume(vol float64) {
	d.mu.Lock()
	d.outputVol = vol
	d.mu.Unlock()
}

// Close stops and releases all audio streams.
func (d *AudioDevice) Close() {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.captureStream != nil {
		if err := d.captureStream.Abort(); err != nil {
			slog.Error("failed to abort capture stream", "err", err)
		}
		d.captureStream = nil
	}

	if d.playbackStream != nil {
		if err := d.playbackStream.Abort(); err != nil {
			slog.Error("failed to abort playback stream", "err", err)
		}
		d.playbackStream = nil
	}
}

func resolveInputDevice(name string) (*portaudio.DeviceInfo, error) {
	if name == "" {
		return portaudio.DefaultInputDevice()
	}
	return resolveDevice(name, true)
}

func resolveOutputDevice(name string) (*portaudio.DeviceInfo, error) {
	if name == "" {
		return portaudio.DefaultOutputDevice()
	}
	return resolveDevice(name, false)
}

func resolveDevice(name string, input bool) (*portaudio.DeviceInfo, error) {
	devices, err := portaudio.Devices()
	if err != nil {
		return nil, err
	}
	return findDeviceByName(devices, name, input)
}

func findDeviceByName(devices []*portaudio.DeviceInfo, name string, input bool) (*portaudio.DeviceInfo, error) {
	requiredChannels := stereo
	capability := "playback"
	if input {
		requiredChannels = channels
		capability = "capture"
	}

	for _, device := range devices {
		if device.Name != name {
			continue
		}
		if input && device.MaxInputChannels < requiredChannels {
			return nil, fmt.Errorf("input device %q does not support %d channel %s", name, requiredChannels, capability)
		}
		if !input && device.MaxOutputChannels < requiredChannels {
			return nil, fmt.Errorf("output device %q does not support %d channel %s", name, requiredChannels, capability)
		}
		return device, nil
	}

	if input {
		return nil, fmt.Errorf("input device %q not found", name)
	}
	return nil, fmt.Errorf("output device %q not found", name)
}
