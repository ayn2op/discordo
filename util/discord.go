package util

import "github.com/ayntgl/discordgo"

// FindMessageByID returns the index and the `*Message` struct of the current message if the given message ID *mID* is equal to the current message ID. If the given message ID *mID* is not found in the given slice *ms*, `-1` and `nil` are returned instead.
func FindMessageByID(ms []*discordgo.Message, mID string) (int, *discordgo.Message) {
	for i, m := range ms {
		if m.ID == mID {
			return i, m
		}
	}

	return -1, nil
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

// HasPermission returns a boolean that indicates whether the client user has the given permission *p* in the given channel ID *cID*.
func HasPermission(s *discordgo.State, cID string, p int64) bool {
	perm, err := s.UserChannelPermissions(s.User.ID, cID)
	if err != nil {
		return false
	}

	return perm&p == p
}
