package voice

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"sync"
	"time"

	ariDiscord "github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/diamondburned/ningen/v3"
	disgoDiscord "github.com/disgoorg/disgo/discord"
	disgoGateway "github.com/disgoorg/disgo/gateway"
	disgoVoice "github.com/disgoorg/disgo/voice"
	"github.com/disgoorg/godave/golibdave"
	"github.com/disgoorg/snowflake/v2"
)

// VoiceState represents the current voice connection state.
type VoiceState int

const (
	VoiceDisconnected VoiceState = iota
	VoiceConnecting
	VoiceConnected
)

// String returns a human-readable name for the voice state.
func (s VoiceState) String() string {
	switch s {
	case VoiceDisconnected:
		return "Disconnected"
	case VoiceConnecting:
		return "Connecting"
	case VoiceConnected:
		return "Connected"
	default:
		return "Unknown"
	}
}

// VoiceManager manages voice connections and audio pipelines using disgo's
// voice package for the actual voice gateway/UDP connection (which supports
// modern encryption modes), while keeping arikawa for the main Discord gateway.
type VoiceManager struct {
	state *ningen.State

	// disgo voice manager and connection
	voiceMgr disgoVoice.Manager
	conn     disgoVoice.Conn

	mu        sync.RWMutex
	connState VoiceState
	guildID   ariDiscord.GuildID
	channelID ariDiscord.ChannelID
	muted     bool
	deafened  bool

	// Audio pipeline
	audio       *AudioDevice
	encoder     *OpusEncoder
	mixer       *Mixer
	audioSender disgoVoice.AudioSender

	// Callbacks
	onStateChange       func(VoiceState)
	onSpeakingChange    func(userID ariDiscord.UserID, speaking bool)
	onParticipantChange func()
	speakingTimers      map[ariDiscord.UserID]*time.Timer

	// Pipeline control
	cancel context.CancelFunc

	// Gateway event cleanup
	detachHandlers []func()

	// Config
	noiseGate     float64
	voiceActivity bool
	inputDevice   string
	outputDevice  string
	inputVol      float64
	outputVol     float64
}

type Config struct {
	InputDevice   string
	OutputDevice  string
	InputVolume   float64
	OutputVolume  float64
	VoiceActivity bool
	NoiseGate     float64
}

// NewVoiceManager creates a new voice manager. The disgo voice manager is
// lazily initialized on the first Join call, since the current user ID is
// not available in the state cabinet until after the READY event.
func NewVoiceManager(state *ningen.State, cfg Config) *VoiceManager {
	return &VoiceManager{
		state:          state,
		noiseGate:      cfg.NoiseGate,
		voiceActivity:  cfg.VoiceActivity,
		inputDevice:    cfg.InputDevice,
		outputDevice:   cfg.OutputDevice,
		inputVol:       cfg.InputVolume,
		outputVol:      cfg.OutputVolume,
		speakingTimers: make(map[ariDiscord.UserID]*time.Timer),
	}
}

// ensureVoiceMgr lazily initializes the disgo voice manager.
func (vm *VoiceManager) ensureVoiceMgr() error {
	if vm.voiceMgr != nil {
		return nil
	}

	me, err := vm.state.Cabinet.Me()
	if err != nil {
		return fmt.Errorf("failed to get current user: %w", err)
	}

	userID := snowflake.ID(me.ID)

	// The StateUpdateFunc bridges disgo's voice state updates to arikawa's
	// main gateway.
	stateUpdateFunc := func(ctx context.Context, guildID snowflake.ID, channelID *snowflake.ID, selfMute bool, selfDeaf bool) error {
		var chID ariDiscord.ChannelID
		if channelID != nil {
			chID = ariDiscord.ChannelID(*channelID)
		} else {
			chID = ariDiscord.ChannelID(ariDiscord.NullSnowflake)
		}

		return vm.state.SendGateway(ctx, &gateway.UpdateVoiceStateCommand{
			GuildID:   ariDiscord.GuildID(guildID),
			ChannelID: chID,
			SelfMute:  selfMute,
			SelfDeaf:  selfDeaf,
		})
	}

	vm.voiceMgr = disgoVoice.NewManager(stateUpdateFunc, userID,
		disgoVoice.WithDaveSessionCreateFunc(golibdave.NewSession),
	)
	return nil
}

// Join connects to a voice channel and starts the audio pipeline.
func (vm *VoiceManager) Join(ctx context.Context, guildID ariDiscord.GuildID, channelID ariDiscord.ChannelID) error {
	if err := vm.ensureVoiceMgr(); err != nil {
		return err
	}

	vm.mu.Lock()
	if vm.connState != VoiceDisconnected {
		vm.mu.Unlock()
		slog.Debug("leaving previous voice channel before joining new one")
		if err := vm.Leave(); err != nil {
			slog.Error("failed to leave previous voice channel", "err", err)
		}
		// Allow the voice gateway to fully tear down before reconnecting.
		time.Sleep(250 * time.Millisecond)
		vm.mu.Lock()
	}

	vm.connState = VoiceConnecting
	vm.guildID = guildID
	vm.channelID = channelID
	vm.mu.Unlock()

	vm.notifyStateChange(VoiceConnecting)

	// Register arikawa gateway handlers to forward voice events to disgo.
	conn := vm.voiceMgr.CreateConn(snowflake.ID(guildID))

	// Log voice gateway events for debugging.
	conn.SetEventHandlerFunc(func(_ disgoVoice.Gateway, op disgoVoice.Opcode, _ int, _ disgoVoice.GatewayMessageData) {
		slog.Debug("voice gateway event", "opcode", op)
	})

	// Forward VoiceStateUpdate events from the arikawa gateway to disgo.
	detach1 := vm.state.AddHandler(func(ev *gateway.VoiceStateUpdateEvent) {
		slog.Debug("forwarding voice state update",
			"user_id", ev.UserID,
			"channel_id", ev.ChannelID,
			"guild_id", ev.GuildID,
			"session_id", ev.SessionID,
		)
		conn.HandleVoiceStateUpdate(disgoGateway.EventVoiceStateUpdate{
			VoiceState: ariVoiceStateToDisgo(ev.VoiceState),
		})
	})

	// Forward VoiceServerUpdate events from the arikawa gateway to disgo.
	detach2 := vm.state.AddHandler(func(ev *gateway.VoiceServerUpdateEvent) {
		endpoint := ev.Endpoint
		conn.HandleVoiceServerUpdate(disgoGateway.EventVoiceServerUpdate{
			Token:    ev.Token,
			GuildID:  snowflake.ID(ev.GuildID),
			Endpoint: &endpoint,
		})
	})

	vm.mu.Lock()
	vm.conn = conn
	vm.detachHandlers = []func(){detach1, detach2}
	vm.mu.Unlock()

	slog.Info("joining voice channel", "guild_id", guildID, "channel_id", channelID)

	// Open the voice connection. This sends the voice state update through
	// the arikawa gateway (via our StateUpdateFunc), waits for the voice
	// server info, connects to the voice gateway with modern encryption,
	// and establishes the UDP audio connection.
	vm.mu.RLock()
	muted := vm.muted
	deafened := vm.deafened
	vm.mu.RUnlock()

	if err := conn.Open(ctx, snowflake.ID(channelID), muted, deafened); err != nil {
		slog.Error("voice connection open failed", "err", err)
		vm.cleanupConn()
		vm.notifyStateChange(VoiceDisconnected)
		return err
	}

	// Initialize audio components.
	encoder, err := NewOpusEncoder()
	if err != nil {
		vm.cleanupConn()
		vm.notifyStateChange(VoiceDisconnected)
		return err
	}

	audio := NewAudioDevice(vm.inputDevice, vm.outputDevice, vm.inputVol, vm.outputVol)
	mixer := NewMixer()

	pipeCtx, cancel := context.WithCancel(context.Background())

	// Create OpusFrameProvider for disgo's AudioSender.
	provider := &micProvider{
		vm:      vm,
		ctx:     pipeCtx,
		audio:   audio,
		encoder: encoder,
	}
	captureCh, err := audio.StartCapture(pipeCtx)
	if err != nil {
		cancel()
		audio.Close()
		vm.cleanupConn()
		vm.notifyStateChange(VoiceDisconnected)
		return fmt.Errorf("failed to start audio capture: %w", err)
	}
	provider.captureCh = captureCh

	playbackCh, err := audio.StartPlayback(pipeCtx)
	if err != nil {
		cancel()
		audio.Close()
		vm.cleanupConn()
		vm.notifyStateChange(VoiceDisconnected)
		return fmt.Errorf("failed to start audio playback: %w", err)
	}

	sender := disgoVoice.NewAudioSender(slog.Default(), provider, conn)

	vm.mu.Lock()
	vm.encoder = encoder
	vm.audio = audio
	vm.mixer = mixer
	vm.audioSender = sender
	vm.cancel = cancel
	vm.connState = VoiceConnected
	vm.mu.Unlock()

	vm.notifyStateChange(VoiceConnected)

	// Set up audio receive via disgo's OpusFrameReceiver.
	conn.SetOpusFrameReceiver(&opusReceiver{vm: vm})

	// Start the audio sender (handles timing, speaking flags, silence frames).
	sender.Open()

	// Start the playback loop that drains the mixer at a steady 20ms rate.
	go vm.playbackLoop(pipeCtx, playbackCh)

	return nil
}

// cleanupConn detaches gateway handlers and removes the disgo connection.
func (vm *VoiceManager) cleanupConn() {
	vm.mu.Lock()
	guildID := vm.guildID
	for _, detach := range vm.detachHandlers {
		detach()
	}
	vm.detachHandlers = nil
	vm.conn = nil
	vm.connState = VoiceDisconnected
	vm.guildID = 0
	vm.channelID = 0
	vm.mu.Unlock()

	if guildID.IsValid() {
		vm.voiceMgr.RemoveConn(snowflake.ID(guildID))
	}
}

// Leave disconnects from the current voice channel.
func (vm *VoiceManager) Leave() error {
	vm.mu.Lock()
	if vm.connState == VoiceDisconnected {
		vm.mu.Unlock()
		return nil
	}

	cancel := vm.cancel
	audio := vm.audio
	mixer := vm.mixer
	sender := vm.audioSender
	guildID := vm.guildID

	vm.cancel = nil
	vm.conn = nil
	vm.audio = nil
	vm.encoder = nil
	vm.mixer = nil
	vm.audioSender = nil
	vm.connState = VoiceDisconnected
	vm.guildID = 0
	vm.channelID = 0

	for _, detach := range vm.detachHandlers {
		detach()
	}
	vm.detachHandlers = nil
	vm.mu.Unlock()

	if sender != nil {
		sender.Close()
	}

	if cancel != nil {
		cancel()
	}

	if audio != nil {
		audio.Close()
	}

	if mixer != nil {
		mixer.Close()
	}

	// RemoveConn closes the connection and removes it from the voice manager.
	// We intentionally do NOT call conn.Close() separately to avoid a
	// double-close race that leaves the gateway in a half-torn-down state.
	vm.voiceMgr.RemoveConn(snowflake.ID(guildID))
	vm.stopSpeakingTimers()
	vm.notifyStateChange(VoiceDisconnected)
	return nil
}

// SetMute sets the mute state and syncs it to Discord.
func (vm *VoiceManager) SetMute(muted bool) error {
	vm.mu.RLock()
	deafened := vm.deafened
	vm.mu.RUnlock()
	return vm.setSelfState(muted, deafened)
}

// SetDeafen sets the deafen state and syncs it to Discord.
func (vm *VoiceManager) SetDeafen(deafened bool) error {
	vm.mu.RLock()
	muted := vm.muted
	vm.mu.RUnlock()
	return vm.setSelfState(muted, deafened)
}

// ToggleMute toggles the mute state and returns the new value.
func (vm *VoiceManager) ToggleMute() (bool, error) {
	vm.mu.RLock()
	muted := !vm.muted
	deafened := vm.deafened
	vm.mu.RUnlock()

	if err := vm.setSelfState(muted, deafened); err != nil {
		return !muted, err
	}
	return muted, nil
}

// ToggleDeafen toggles the deafen state and returns the new value.
func (vm *VoiceManager) ToggleDeafen() (bool, error) {
	vm.mu.RLock()
	muted := vm.muted
	deafened := !vm.deafened
	vm.mu.RUnlock()

	if err := vm.setSelfState(muted, deafened); err != nil {
		return !deafened, err
	}
	return deafened, nil
}

// State returns the current connection state.
func (vm *VoiceManager) State() VoiceState {
	vm.mu.RLock()
	defer vm.mu.RUnlock()
	return vm.connState
}

// ChannelID returns the current voice channel ID.
func (vm *VoiceManager) ChannelID() ariDiscord.ChannelID {
	vm.mu.RLock()
	defer vm.mu.RUnlock()
	return vm.channelID
}

// GuildID returns the current guild ID.
func (vm *VoiceManager) GuildID() ariDiscord.GuildID {
	vm.mu.RLock()
	defer vm.mu.RUnlock()
	return vm.guildID
}

// IsMuted returns whether the user is muted.
func (vm *VoiceManager) IsMuted() bool {
	vm.mu.RLock()
	defer vm.mu.RUnlock()
	return vm.muted
}

// IsDeafened returns whether the user is deafened.
func (vm *VoiceManager) IsDeafened() bool {
	vm.mu.RLock()
	defer vm.mu.RUnlock()
	return vm.deafened
}

// OnStateChange registers a callback for voice state changes.
func (vm *VoiceManager) OnStateChange(f func(VoiceState)) {
	vm.mu.Lock()
	vm.onStateChange = f
	vm.mu.Unlock()
}

// OnSpeakingChange registers a callback for user speaking state changes.
func (vm *VoiceManager) OnSpeakingChange(f func(ariDiscord.UserID, bool)) {
	vm.mu.Lock()
	vm.onSpeakingChange = f
	vm.mu.Unlock()
}

// OnParticipantChange registers a callback for participant join/leave events.
func (vm *VoiceManager) OnParticipantChange(f func()) {
	vm.mu.Lock()
	vm.onParticipantChange = f
	vm.mu.Unlock()
}

// Close cleans up all voice resources.
func (vm *VoiceManager) Close() error {
	err := vm.Leave()
	if vm.voiceMgr != nil {
		vm.voiceMgr.Close(context.Background())
	}
	return err
}

func (vm *VoiceManager) setSelfState(muted bool, deafened bool) error {
	vm.mu.RLock()
	guildID := vm.guildID
	channelID := vm.channelID
	state := vm.connState
	vm.mu.RUnlock()

	if state == VoiceDisconnected || !guildID.IsValid() || !channelID.IsValid() {
		return errors.New("voice is not connected")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := vm.state.SendGateway(ctx, &gateway.UpdateVoiceStateCommand{
		GuildID:   guildID,
		ChannelID: channelID,
		SelfMute:  muted,
		SelfDeaf:  deafened,
	}); err != nil {
		return err
	}

	vm.mu.Lock()
	vm.muted = muted
	vm.deafened = deafened
	vm.mu.Unlock()
	return nil
}

// micProvider implements disgoVoice.OpusFrameProvider, feeding mic audio into
// disgo's AudioSender which handles RTP timing, speaking flags, and silence.
type micProvider struct {
	vm        *VoiceManager
	ctx       context.Context
	audio     *AudioDevice
	encoder   *OpusEncoder
	captureCh <-chan []int16
}

func (p *micProvider) ProvideOpusFrame() ([]byte, error) {
	if p.captureCh == nil {
		return nil, io.EOF
	}

	select {
	case <-p.ctx.Done():
		return nil, io.EOF
	case pcm, ok := <-p.captureCh:
		if !ok {
			return nil, io.EOF
		}

		p.vm.mu.RLock()
		muted := p.vm.muted
		p.vm.mu.RUnlock()

		if muted || p.vm.isSilent(pcm) {
			// Return empty slice = silence (AudioSender handles speaking flags).
			return nil, nil
		}

		opusData, err := p.encoder.Encode(pcm)
		if err != nil {
			slog.Error("failed to encode pcm", "err", err)
			return nil, nil
		}

		return opusData, nil
	}
}

func (p *micProvider) Close() {
	// Audio device cleanup is handled by VoiceManager.Leave().
}

// playbackLoop drains the mixer at a steady 20ms rate and sends mixed PCM
// to the speaker. Running on a fixed ticker avoids the echo/repeat artifacts
// caused by draining on every received packet.
func (vm *VoiceManager) playbackLoop(ctx context.Context, playbackCh chan<- []int16) {
	vm.mu.RLock()
	mixer := vm.mixer
	vm.mu.RUnlock()

	if mixer == nil {
		return
	}

	ticker := time.NewTicker(20 * time.Millisecond)
	defer ticker.Stop()
	cleanupTicker := time.NewTicker(staleTimeout)
	defer cleanupTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-cleanupTicker.C:
			mixer.Cleanup()
		case <-ticker.C:
			vm.mu.RLock()
			deafened := vm.deafened
			vm.mu.RUnlock()

			if deafened {
				continue
			}

			mixed := mixer.Mix()
			select {
			case playbackCh <- mixed:
			default:
			}
		}
	}
}

// opusReceiver implements disgoVoice.OpusFrameReceiver to receive audio from
// other users and feed it into the mixer for playback.
type opusReceiver struct {
	vm *VoiceManager
}

func (r *opusReceiver) ReceiveOpusFrame(userID snowflake.ID, packet *disgoVoice.Packet) error {
	r.vm.mu.RLock()
	deafened := r.vm.deafened
	mixer := r.vm.mixer
	r.vm.mu.RUnlock()

	if deafened || mixer == nil {
		return nil
	}

	if err := mixer.AddPacket(packet.SSRC, packet.Opus); err != nil {
		slog.Error("failed to add packet to mixer", "ssrc", packet.SSRC, "err", err)
	}

	r.vm.markSpeaking(ariDiscord.UserID(userID))

	return nil
}

func (r *opusReceiver) CleanupUser(userID snowflake.ID) {
	r.vm.clearSpeaking(ariDiscord.UserID(userID))
}

func (r *opusReceiver) Close() {
	r.vm.mu.RLock()
	mixer := r.vm.mixer
	r.vm.mu.RUnlock()

	if mixer != nil {
		mixer.Close()
	}
}

func (vm *VoiceManager) isSilent(pcm []int16) bool {
	if !vm.voiceActivity || vm.noiseGate <= 0 {
		return false
	}

	var sum float64
	for _, s := range pcm {
		sum += float64(s) * float64(s)
	}
	rms := sum / float64(len(pcm))

	return rms < vm.noiseGate*vm.noiseGate
}

func (vm *VoiceManager) notifyStateChange(s VoiceState) {
	vm.mu.RLock()
	cb := vm.onStateChange
	vm.mu.RUnlock()

	if cb != nil {
		cb(s)
	}
}

func (vm *VoiceManager) markSpeaking(userID ariDiscord.UserID) {
	const speakingHold = 250 * time.Millisecond

	vm.mu.Lock()
	if timer, ok := vm.speakingTimers[userID]; ok {
		timer.Reset(speakingHold)
		vm.mu.Unlock()
		return
	}

	vm.speakingTimers[userID] = time.AfterFunc(speakingHold, func() {
		vm.clearSpeaking(userID)
	})
	cb := vm.onSpeakingChange
	vm.mu.Unlock()

	if cb != nil {
		cb(userID, true)
	}
}

func (vm *VoiceManager) clearSpeaking(userID ariDiscord.UserID) {
	vm.mu.Lock()
	if timer, ok := vm.speakingTimers[userID]; ok {
		timer.Stop()
		delete(vm.speakingTimers, userID)
		cb := vm.onSpeakingChange
		vm.mu.Unlock()
		if cb != nil {
			cb(userID, false)
		}
		return
	}
	vm.mu.Unlock()
}

func (vm *VoiceManager) stopSpeakingTimers() {
	vm.mu.Lock()
	timers := vm.speakingTimers
	vm.speakingTimers = make(map[ariDiscord.UserID]*time.Timer)
	vm.mu.Unlock()

	for _, timer := range timers {
		timer.Stop()
	}
}

// ariVoiceStateToDisgo converts an arikawa VoiceState to a disgo VoiceState
// for forwarding gateway events.
func ariVoiceStateToDisgo(vs ariDiscord.VoiceState) disgoDiscord.VoiceState {
	var chID *snowflake.ID
	if vs.ChannelID.IsValid() {
		id := snowflake.ID(vs.ChannelID)
		chID = &id
	}
	return disgoDiscord.VoiceState{
		GuildID:   snowflake.ID(vs.GuildID),
		ChannelID: chID,
		UserID:    snowflake.ID(vs.UserID),
		SessionID: vs.SessionID,
		GuildDeaf: vs.Deaf,
		GuildMute: vs.Mute,
		SelfDeaf:  vs.SelfDeaf,
		SelfMute:  vs.SelfMute,
		SelfVideo: vs.SelfVideo,
		Suppress:  vs.Suppress,
	}
}
