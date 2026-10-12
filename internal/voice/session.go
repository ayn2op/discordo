//go:build !no_voice

package voice

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"log/slog"
	"math"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ayn2op/arikawa/v3/discord"
	"github.com/ayn2op/arikawa/v3/gateway"
	"github.com/ayn2op/arikawa/v3/state"
	"github.com/ayn2op/arikawa/v3/utils/ws"
	"github.com/ayn2op/arikawa/v3/voice"
	"github.com/ayn2op/arikawa/v3/voice/udp"
	"github.com/ayn2op/arikawa/v3/voice/voicegateway"
	"github.com/gen2brain/malgo"
	dave "github.com/thomas-vilte/dave-go/session"
	"github.com/tphakala/go-opus/opus"
)

const (
	timeout = 15 * time.Second
	// hangover is how many frames are still sent after the last that was loud enough, for the ends of words not to be cut off.
	hangover = 15
)

// audio is what the audio devices are opened with. It is of the first backend that works but the null one, which malgo would otherwise silently use when the system has no audio library.
var audio = sync.OnceValues(func() (*malgo.AllocatedContext, error) {
	return malgo.InitContext([]malgo.Backend{
		malgo.BackendWasapi, malgo.BackendDsound, malgo.BackendWinmm,
		malgo.BackendCoreaudio,
		malgo.BackendSndio, malgo.BackendAudio4, malgo.BackendOss,
		malgo.BackendPulseaudio, malgo.BackendAlsa, malgo.BackendJack,
	}, malgo.ContextConfig{}, nil)
})

// startDevice starts a playback or capture device, which calls data with what to fill or with what was captured.
func startDevice(kind malgo.DeviceType, channels int, data func(output, input []byte)) (*malgo.Device, error) {
	ctx, err := audio()
	if err != nil {
		return nil, err
	}

	config := malgo.DefaultDeviceConfig(kind)
	config.SampleRate = sampleRate
	config.PeriodSizeInMilliseconds = 10
	config.Playback.Format, config.Playback.Channels = malgo.FormatS16, uint32(channels)
	config.Capture.Format, config.Capture.Channels = malgo.FormatS16, uint32(channels)

	device, err := malgo.InitDevice(ctx.Context, config, malgo.DeviceCallbacks{
		Data: func(output, input []byte, _ uint32) { data(output, input) },
	})
	if err != nil {
		return nil, err
	}

	if err := device.Start(); err != nil {
		device.Uninit()
		return nil, err
	}

	return device, nil
}

// Session is the voice channel that the user is in, if any.
type Session struct {
	state *state.State
	// threshold is how loud a sample of the microphone must be for what is said to be sent.
	threshold int
	// updates is sent to when the status or who is speaking changes.
	updates chan struct{}
	// mic is sent what the microphone captures.
	mic chan []byte

	// mu is held while joining, leaving and muting, which are not done while another is.
	mu sync.Mutex
	// session is nil until the first voice channel is joined.
	session atomic.Pointer[voice.Session]
	// playback plays while a voice channel is joined, and capture captures while the user is not muted in it.
	playback, capture *malgo.Device

	stateMu  sync.Mutex
	status   Status
	speaking map[discord.UserID]bool
	speakers map[uint32]*speaker
}

// New returns a Session that sends what the microphone captures louder than sensitivity, in dB.
func New(state *state.State, sensitivity float64) *Session {
	s := &Session{
		state:     state,
		threshold: int(math.Pow(10, sensitivity/20) * math.MaxInt16),
		updates:   make(chan struct{}, 1),
		mic:       make(chan []byte, 4),
		speaking:  make(map[discord.UserID]bool),
		speakers:  make(map[uint32]*speaker),
	}
	state.AddHandler(s.follow)

	return s
}

// follow keeps up with the user being moved to another voice channel or disconnected, by someone else or by connecting from another client.
func (s *Session) follow(event *gateway.VoiceStateUpdateEvent) {
	me, _ := s.state.Cabinet.Me()
	if me == nil || event.UserID != me.ID {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	status, own := s.Status(), event.SessionID == s.state.Ready().SessionID
	switch {
	case !status.Joined || own && event.ChannelID == status.ChannelID:
	case own && event.ChannelID.IsValid():
		s.setStatus(Status{ChannelID: event.ChannelID, Joined: true})
	default:
		s.left()
	}
}

// Updates is sent to when the status or who is speaking changes.
func (s *Session) Updates() <-chan struct{} {
	return s.updates
}

func (s *Session) Status() Status {
	s.stateMu.Lock()
	defer s.stateMu.Unlock()

	return s.status
}

// Speaking reports whether the user is speaking in the voice channel joined.
func (s *Session) Speaking(userID discord.UserID) bool {
	s.stateMu.Lock()
	defer s.stateMu.Unlock()

	return s.speaking[userID]
}

// Latency returns how long the voice server of the voice channel joined takes to answer, or 0 if it is not known.
func (s *Session) Latency() time.Duration {
	if session := s.session.Load(); session != nil {
		return session.Latency()
	}

	return 0
}

// update changes the state with f and tells Updates if f reports that it did.
func (s *Session) update(f func() bool) {
	s.stateMu.Lock()
	changed := f()
	s.stateMu.Unlock()

	if changed {
		select {
		case s.updates <- struct{}{}:
		default:
		}
	}
}

// setStatus changes the status, which whoever spoke and was heard does not outlast.
func (s *Session) setStatus(status Status) {
	s.update(func() bool {
		s.status = status
		clear(s.speaking)
		clear(s.speakers)
		return true
	})
}

func (s *Session) setSpeaking(userID discord.UserID, speaking bool) {
	s.update(func() bool {
		changed := s.speaking[userID] != speaking
		s.speaking[userID] = speaking
		return changed
	})
}

// setDevice starts device, which then calls data, if on, and stops it otherwise.
func setDevice(device **malgo.Device, on bool, kind malgo.DeviceType, data func(output, input []byte)) (err error) {
	switch {
	case on && *device == nil:
		*device, err = startDevice(kind, channels, data)
	case !on && *device != nil:
		(*device).Uninit()
		*device = nil
	}

	return err
}

// setDevices captures what the user says unless mute, and plays what is said unless deafen.
func (s *Session) setDevices(mute, deafen bool) error {
	err := errors.Join(
		setDevice(&s.capture, !mute, malgo.Capture, func(_, input []byte) { s.queue(bytes.Clone(input)) }),
		setDevice(&s.playback, !deafen, malgo.Playback, s.mix),
	)
	if mute {
		// Silence makes what is being said end.
		s.queue(make([]byte, 2*hangover*frameBytes))
	}

	return err
}

// left stops the devices, in no voice channel.
func (s *Session) left() {
	s.setDevices(true, true)
	s.setStatus(Status{})
}

// Join joins the voice channel, muted, leaving the one that the user is in.
func (s *Session) Join(channelID discord.ChannelID) (err error) {
	if !s.mu.TryLock() {
		return nil
	}
	defer s.mu.Unlock()

	s.setStatus(Status{ChannelID: channelID})

	defer func() {
		// Discord may have the user in the voice channel regardless, and joining it again would then do nothing.
		if err != nil {
			s.leave()
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	session := s.session.Load()
	if session == nil {
		if session, err = voice.NewSession(s.state, s.state.Handler); err != nil {
			return err
		}

		session.DAVE = dave.CreateFunc()
		session.AddHandler(func(event *ws.BackgroundErrorEvent) { slog.Debug("voice gateway error", "err", event.Err) })
		s.session.Store(session)

		go s.receive(session)
		go s.send(session)
	}

	if err := s.setDevices(true, false); err != nil {
		return err
	}

	if err := session.JoinChannel(ctx, channelID, true, false); err != nil {
		return err
	}

	// The voice server sends nothing before it is told this.
	if err := session.Speaking(ctx, voicegateway.Microphone); err != nil {
		return err
	}

	s.setStatus(Status{ChannelID: channelID, Joined: true})

	return nil
}

// Leave leaves the voice channel that the user is in, if any.
func (s *Session) Leave() error {
	if !s.mu.TryLock() {
		return nil
	}
	defer s.mu.Unlock()

	return s.leave()
}

func (s *Session) leave() error {
	defer s.left()

	session := s.session.Load()
	if session == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	return session.Leave(ctx)
}

// Toggle mutes or unmutes the user in the voice channel joined, if any, or deafens or undeafens them if deafen. Deafening also mutes, and unmuting also undeafens.
func (s *Session) Toggle(deafen bool) error {
	if !s.mu.TryLock() {
		return nil
	}
	defer s.mu.Unlock()

	status := s.Status()
	channel, err := s.state.Cabinet.Channel(status.ChannelID)
	if err != nil || !status.Joined {
		return nil
	}

	mute, deafen := deafen || s.capture != nil, deafen && s.playback != nil
	if err := s.setDevices(mute, deafen); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	return s.state.SendGateway(ctx, &gateway.UpdateVoiceStateCommand{
		GuildID:   channel.GuildID,
		ChannelID: channel.ID,
		SelfMute:  mute,
		SelfDeaf:  deafen,
	})
}

// queue queues what was captured to be sent, in place of the oldest if what is queued is not being sent fast enough, for what is said not to be sent ever later.
func (s *Session) queue(pcm []byte) {
	for {
		select {
		case s.mic <- pcm:
			return
		default:
		}

		select {
		case <-s.mic:
		default:
		}
	}
}

// mix fills output with what everyone says. It is what the playback device calls.
func (s *Session) mix(output, _ []byte) {
	s.stateMu.Lock()
	defer s.stateMu.Unlock()

	for _, speaker := range s.speakers {
		speaker.mix(output)
	}
}

// receive buffers what each user says in the voice channels that session joins.
func (s *Session) receive(session *voice.Session) {
	for {
		packet, err := session.ReadPacket()
		if errors.Is(err, udp.ErrDecryptionFailed) {
			continue
		}

		if err != nil {
			// In no voice channel, or joining one.
			time.Sleep(100 * time.Millisecond)
			continue
		}

		s.stateMu.Lock()
		speaker := s.speakers[packet.SSRC()]
		if speaker == nil {
			if speaker, err = newSpeaker(packet.Sequence()); err != nil {
				s.stateMu.Unlock()
				slog.Error("failed to create Opus decoder", "err", err)
				return
			}

			s.speakers[packet.SSRC()] = speaker
		}

		s.stateMu.Unlock()

		if err := speaker.write(packet.Sequence(), packet.Opus); err != nil {
			slog.Debug("failed to decode voice packet", "err", err, "ssrc", packet.SSRC())
		}

		s.setSpeaking(session.User(packet.SSRC()), string(packet.Opus) != voice.Silence)
	}
}

// send sends what the microphone captures to the voice channels that session joins, while it is loud enough.
func (s *Session) send(session *voice.Session) {
	encoder, err := opus.NewEncoder(opus.EncoderConfig{SampleRate: sampleRate, Channels: channels})
	if err != nil {
		slog.Error("failed to create Opus encoder", "err", err)
		return
	}

	me, _ := s.state.Cabinet.Me()

	var (
		pcm     []byte
		samples = make([]int16, frameSamples)
		packet  = make([]byte, 4000)
		silence = []byte(voice.Silence)
		// quiet is how many frames ago the last that was loud enough was.
		quiet = hangover
	)

	for captured := range s.mic {
		for pcm = append(pcm, captured...); len(pcm) >= frameBytes; pcm = pcm[frameBytes:] {
			quiet++
			for i := range samples {
				samples[i] = int16(binary.LittleEndian.Uint16(pcm[2*i:]))
				if int(samples[i]) >= s.threshold || -int(samples[i]) >= s.threshold {
					quiet = 0
				}
			}

			var err error
			switch {
			case quiet < hangover:
				var n int
				if n, err = encoder.Encode(samples, packet); err == nil {
					_, err = session.Write(packet[:n])
				}
			case quiet < hangover+5:
				// A few frames of silence are for what was said not to be interpolated with what is said next.
				_, err = session.Write(silence)
			}

			if err != nil {
				slog.Debug("failed to send voice packet", "err", err)
			}

			if me != nil {
				s.setSpeaking(me.ID, quiet < hangover)
			}
		}
	}
}
