package ui

import (
	"strings"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/state"
)

func channelToString(c discord.Channel) string {
	var repr string

	switch c.Type {
	case discord.GuildText:
		repr = "#" + c.Name
	case discord.DirectMessage:
		rp := c.DMRecipients[0]
		repr = rp.Username + "#" + rp.Discriminator
	case discord.GroupDM:
		repr = c.Name
		// if the name wasn't loaded, use it as a backup
		if repr == "" {
			rps := make([]string, len(c.DMRecipients))
			for i, r := range c.DMRecipients {
				rps[i] = r.Username + "#" + r.Discriminator
			}

			repr = strings.Join(rps, ", ")
		}
	default:
		repr = c.Name
	}

	return repr
}

func findMessageByID(ms []discord.Message, mID discord.MessageID) (int, *discord.Message) {
	for i, m := range ms {
		if m.ID == mID {
			return i, &m
		}
	}

	return -1, nil
}

func channelIsInDMCategory(c *discord.Channel) bool {
	return c.Type == discord.DirectMessage || c.Type == discord.GroupDM
}

func hasPermission(s *state.State, cID discord.ChannelID, p discord.Permissions) bool {
	perm, err := s.Permissions(cID, s.Ready().User.ID)
	if err != nil {
		return false
	}

	return perm&p == p
}
