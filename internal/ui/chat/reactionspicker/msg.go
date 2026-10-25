package reactionspicker

import "github.com/ayn2op/arikawa/v3/discord"

// ShowMsg lists the emojis to react to the message with MessageID in ChannelID with, in GuildID.
type ShowMsg struct {
	GuildID   discord.GuildID
	ChannelID discord.ChannelID
	MessageID discord.MessageID
}

// SelectedMsg asks to react with Emoji to the message with MessageID in ChannelID.
type SelectedMsg struct {
	ChannelID discord.ChannelID
	MessageID discord.MessageID
	Emoji     discord.APIEmoji
}

type CancelMsg struct{}
