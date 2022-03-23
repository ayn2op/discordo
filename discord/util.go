package discord

import (
	"strings"

	"github.com/ayntgl/astatine"
)

func ChannelToString(c *astatine.Channel) string {
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

func FindMessageByID(ms []*astatine.Message, mID string) (int, *astatine.Message) {
	for i, m := range ms {
		if m.ID == mID {
			return i, m
		}
	}

	return -1, nil
}

func HasPermission(s *astatine.State, cID string, p int64) bool {
	perm, err := s.UserChannelPermissions(s.User.ID, cID)
	if err != nil {
		return false
	}

	return perm&p == p
}
