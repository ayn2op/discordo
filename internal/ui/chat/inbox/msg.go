package inbox

import "github.com/ayn2op/arikawa/v3/discord"

// SelectedMsg asks to open the channel of the row selected.
type SelectedMsg struct {
	ChannelID discord.ChannelID
}

// CancelMsg closes the inbox without opening a channel.
type CancelMsg struct{}
