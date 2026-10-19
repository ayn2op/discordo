package emojipicker

import "github.com/ayn2op/arikawa/v3/discord"

type (
	SelectedMsg struct {
		Emoji discord.APIEmoji
	}
	CancelMsg struct{}
)
