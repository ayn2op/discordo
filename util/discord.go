package util

import (
	"strings"

	"github.com/ayntgl/discordgo"
)

// ChannelToString constructs a string representation of the given channel. The string representation may vary for different channel types.
func ChannelToString(c *discordgo.Channel) string {
	var repr string
	if c.Name != "" {
		repr = "#" + c.Name
	} else if len(c.Recipients) == 1 {
		rp := c.Recipients[0]
		repr = rp.Username + "#" + rp.Discriminator
	} else {
		rps := make([]string, len(c.Recipients))
		for i, r := range c.Recipients {
			rps[i] = r.Username + "#" + r.Discriminator
		}

		repr = strings.Join(rps, ", ")
	}

	return repr
}

// ChannelIsUnread returns `true` if the given channel is marked as read by the client user, otherwise `false`.
func ChannelIsUnread(s *discordgo.State, c *discordgo.Channel) bool {
	if c.LastMessageID == "" {
		return false
	}

	for _, rs := range s.ReadState {
		if c.ID == rs.ID {
			return c.LastMessageID != rs.LastMessageID
		}
	}

	return false
}

// FindMessageByID returns the index and the `*Message` struct of the current message if the given message ID *mID* is equal to the current message ID. If the given message ID *mID* is not found in the given slice *ms*, `-1` and `nil` are returned instead.
func FindMessageByID(ms []*discordgo.Message, mID string) (int, *discordgo.Message) {
	for i, m := range ms {
		if m.ID == mID {
			return i, m
		}
	}

	return -1, nil
}
