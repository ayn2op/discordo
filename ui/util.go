package ui

import (
	"strings"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/state"
)

func channelToString(c discord.Channel) string {
	var repr string
	if c.Name != "" {
		repr = "#" + c.Name
	} else if len(c.DMRecipients) == 1 {
		rp := c.DMRecipients[0]
		repr = rp.Username + "#" + rp.Discriminator
	} else {
		rps := make([]string, len(c.DMRecipients))
		for i, r := range c.DMRecipients {
			rps[i] = r.Username + "#" + r.Discriminator
		}

		repr = strings.Join(rps, ", ")
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

func hasPermission(s *state.State, cID discord.ChannelID, p discord.Permissions) bool {
	perm, err := s.Permissions(cID, s.Ready().User.ID)
	if err != nil {
		return false
	}

	return perm&p == p
}
