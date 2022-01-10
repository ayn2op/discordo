package util

import (
	"strings"

	"github.com/ayntgl/discordgo"
)

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

func HasKeybinding(ks []string, k string) bool {
	for _, repr := range ks {
		if repr == k {
			return true
		}
	}

	return false
}
