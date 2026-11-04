package inbox

import "github.com/ayn2op/arikawa/v3/discord"

// SelectedMsg asks to open the channel of the row selected, on the message after LastReadID, which is its first unread.
type SelectedMsg struct {
	ChannelID  discord.ChannelID
	LastReadID discord.MessageID
}

// CancelMsg closes the inbox without opening a channel.
type CancelMsg struct{}
