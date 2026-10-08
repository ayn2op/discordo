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
func (*Session) ToggleMute() error            { return nil }
func (*Session) ToggleDeafen() error          { return nil }

func (*Session) Join(discord.ChannelID) error {
	return errors.New("discordo was built without voice")
}
