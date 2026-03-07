package voice

import (
	"strings"
	"sync"
	"testing"
	"time"

	ariDiscord "github.com/diamondburned/arikawa/v3/discord"
	"github.com/gordonklaus/portaudio"
)

func TestIsSilentHonorsVoiceActivity(t *testing.T) {
	vm := NewVoiceManager(nil, Config{
		VoiceActivity: true,
		NoiseGate:     0.5,
	})

	pcm := make([]int16, frameSize)
	if !vm.isSilent(pcm) {
		t.Fatal("expected silence when voice activity is enabled")
	}

	vm.voiceActivity = false
	if vm.isSilent(pcm) {
		t.Fatal("expected silence gating to be disabled when voice activity is off")
	}
}

func TestFindDeviceByName(t *testing.T) {
	devices := []*portaudio.DeviceInfo{
		{Name: "Mic", MaxInputChannels: 1},
		{Name: "Speaker", MaxOutputChannels: 2},
	}

	device, err := findDeviceByName(devices, "Mic", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if device.Name != "Mic" {
		t.Fatalf("got device %q", device.Name)
	}

	if _, err := findDeviceByName(devices, "Mic", false); err == nil || !strings.Contains(err.Error(), "output device") {
		t.Fatalf("expected output capability error, got %v", err)
	}

	if _, err := findDeviceByName(devices, "Missing", true); err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("expected not found error, got %v", err)
	}
}

func TestMixerCleanupRemovesStaleDecoders(t *testing.T) {
	stale := time.Now().Add(-staleTimeout - time.Millisecond)
	fresh := time.Now()

	mixer := &Mixer{
		decoders: map[uint32]*decoderState{
			1: {decoder: &OpusDecoder{}, lastSeen: stale},
			2: {decoder: &OpusDecoder{}, lastSeen: fresh},
		},
	}

	mixer.Cleanup()

	if _, ok := mixer.decoders[1]; ok {
		t.Fatal("expected stale decoder to be removed")
	}
	if _, ok := mixer.decoders[2]; !ok {
		t.Fatal("expected fresh decoder to remain")
	}
}

func TestMarkSpeakingExpires(t *testing.T) {
	vm := NewVoiceManager(nil, Config{})

	var (
		mu     sync.Mutex
		events []bool
		done   = make(chan struct{})
	)

	vm.OnSpeakingChange(func(_ ariDiscord.UserID, speaking bool) {
		mu.Lock()
		events = append(events, speaking)
		if len(events) == 2 {
			select {
			case <-done:
			default:
				close(done)
			}
		}
		mu.Unlock()
	})

	vm.markSpeaking(ariDiscord.UserID(42))

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for speaking expiration")
	}

	mu.Lock()
	defer mu.Unlock()
	if len(events) != 2 {
		t.Fatalf("got %d events, want 2", len(events))
	}
	if !events[0] || events[1] {
		t.Fatalf("got events %v, want [true false]", events)
	}
}
