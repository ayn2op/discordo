// Package voice joins voice channels, plays what is said in them and sends what the user says. Built with the no_voice tag, it does none of that and needs no C compiler.
package voice

import "github.com/ayn2op/arikawa/v3/discord"

// Status is what a Session is doing.
type Status struct {
	// ChannelID is the voice channel joined or being joined, if any.
	ChannelID discord.ChannelID
	Joined    bool
}
