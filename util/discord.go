package util

import (
	"github.com/ayntgl/discordgo"
)

func FindMessageByID(ms []*discordgo.Message, mID string) (int, *discordgo.Message) {
	for i, m := range ms {
		if m.ID == mID {
			return i, m
		}
	}

	return -1, nil
}

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

func HasPermission(s *discordgo.State, cID string, p int64) bool {
	perm, err := s.UserChannelPermissions(s.User.ID, cID)
	if err != nil {
		return false
	}

	return perm&p == p
}
