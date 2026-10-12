// Package voice joins voice channels, plays what is said in them, sends what the user says, and plays voice messages. Built with the no_voice tag, it does none of that and needs no C compiler.
package voice

import (
	"time"

	"github.com/ayn2op/arikawa/v3/discord"
)

// Status is what a Session is doing.
type Status struct {
	// ChannelID is the voice channel joined or being joined, if any.
	ChannelID discord.ChannelID
	Joined    bool
}

// PlayerStatus is the voice message a Player is playing or has paused, if ID is valid.
type PlayerStatus struct {
	ID       discord.AttachmentID
	Position time.Duration
	Playing  bool
}
