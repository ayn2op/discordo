//go:build no_voice

package voice

import (
	"errors"
	"time"

	"github.com/ayn2op/arikawa/v3/discord"
	"github.com/ayn2op/arikawa/v3/state"
)

// Session joins no voice channel.
type Session struct{}

func New(*state.State, float64) *Session { return &Session{} }

func (*Session) Updates() <-chan struct{}     { return nil }
func (*Session) Status() Status               { return Status{} }
func (*Session) Speaking(discord.UserID) bool { return false }
func (*Session) Latency() time.Duration       { return 0 }
func (*Session) Leave() error                 { return nil }
func (*Session) Toggle(bool) error            { return nil }

var errNoVoice = errors.New("built without voice (build tag: no_voice)")

func (*Session) Join(discord.ChannelID) error { return errNoVoice }

// Player plays nothing.
type Player struct{}

func NewPlayer() *Player { return new(Player) }

func (*Player) Updates() <-chan struct{}                  { return nil }
func (*Player) Status() PlayerStatus                      { return PlayerStatus{} }
func (*Player) Toggle(discord.AttachmentID) (bool, error) { return false, nil }
func (*Player) Play(discord.AttachmentID, []byte) error   { return errNoVoice }
func (*Player) Seek(time.Duration) error                  { return nil }
