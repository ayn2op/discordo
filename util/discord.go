package util

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/rigormorrtiss/discordgo"
	"github.com/rivo/tview"
)

// WriteMessage parses, renders, and writes a message to the given TextView.
func WriteMessage(v *tview.TextView, m *discordgo.Message, clientID string) {
	var b strings.Builder

	switch m.Type {
	case discordgo.MessageTypeDefault, discordgo.MessageTypeReply:
		// Define a new region and assign message ID as the region ID.
		// Learn more: https://pkg.go.dev/github.com/rivo/tview#hdr-Regions_and_Highlights
		b.WriteString("[\"")
		b.WriteString(m.ID)
		b.WriteString("\"]")
		// Render the message associated with crosspost, channel follow add, pin, or a reply.
		if rm := m.ReferencedMessage; rm != nil {
			b.WriteString(" ╭ ")
			b.WriteString("[::d]")
			parseAuthor(&b, rm.Author, clientID)

			if rm.Content != "" {
				rm.Content = parseMessageMentions(rm.Content, rm.Mentions, clientID)
				b.WriteString(rm.Content)
			}

			b.WriteString("[::-]\n")
		}
		// Render the author of the message.
		parseAuthor(&b, m.Author, clientID)
		// If the message content is not empty, parse the message mentions (users mentioned in the message) and render the message content.
		if m.Content != "" {
			m.Content = parseMessageMentions(m.Content, m.Mentions, clientID)
			b.WriteString(m.Content)
		}
		// If the edited timestamp of the message is not empty; it implies that the message has been edited, hence render the message with edited label for distinction
		if m.EditedTimestamp != "" {
			b.WriteString(" [::d](edited)[::-]")
		}
		// TODO: render message embeds
		for range m.Embeds {
			b.WriteString("\n<EMBED>")
		}
		// Render the message attachments (attached files to the message).
		for _, a := range m.Attachments {
			b.WriteString("\n[")
			b.WriteString(a.Filename)
			b.WriteString("]: ")
			b.WriteString(a.URL)
		}

		fmt.Fprintln(v, b.String())
	case discordgo.MessageTypeGuildMemberJoin:
		b.WriteString("[#5865F2]")
		b.WriteString(m.Author.Username)
		b.WriteString("[-]")
		b.WriteString(" joined the server")

		fmt.Fprintln(v, b.String())
	}
}

func parseMessageMentions(content string, mentions []*discordgo.User, clientID string) string {
	for _, mUser := range mentions {
		var color string
		if mUser.ID == clientID {
			color = "[:#5865F2]"
		} else {
			color = "[#EB459E]"
		}

		content = strings.NewReplacer(
			// <@USER_ID>
			"<@"+mUser.ID+">",
			color+"@"+mUser.Username+"[-:-]",
			// <@!USER_ID>
			"<@!"+mUser.ID+">",
			color+"@"+mUser.Username+"[-:-]",
		).Replace(content)
	}

	return content
}

func parseAuthor(b *strings.Builder, u *discordgo.User, clientID string) {
	// If the message author is the client, modify the text color for distinction.
	if u.ID == clientID {
		b.WriteString("[#57F287]")
	} else {
		b.WriteString("[#ED4245]")
	}

	b.WriteString(u.Username)
	b.WriteString("[-] ")
	// If the message author is a bot account, render the message with bot label for distinction.
	if u.Bot {
		b.WriteString("[#EB459E]BOT[-] ")
	}
}

type loginResponse struct {
	MFA    bool   `json:"mfa"`
	SMS    bool   `json:"sms"`
	Ticket string `json:"ticket"`
	Token  string `json:"token"`
}

// Login creates a new request to the `/login` endpoint for essential login information.
func Login(s *discordgo.Session, email, password string) (*loginResponse, error) {
	data := struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}{email, password}
	resp, err := s.RequestWithBucketID("POST", discordgo.EndpointLogin, data, discordgo.EndpointLogin)
	if err != nil {
		return nil, err
	}

	var lr loginResponse
	err = json.Unmarshal(resp, &lr)
	if err != nil {
		return nil, err
	}

	return &lr, nil
}

// TOTP creates a new request to `/mfa/totp` endpoint for time-based one-time
// passcode for essential login information
func TOTP(s *discordgo.Session, code, ticket string) (*loginResponse, error) {
	data := struct {
		Code   string `json:"code"`
		Ticket string `json:"ticket"`
	}{code, ticket}
	e := discordgo.EndpointAuth + "mfa/totp"
	resp, err := s.RequestWithBucketID("POST", e, data, e)
	if err != nil {
		return nil, err
	}

	var lr loginResponse
	err = json.Unmarshal(resp, &lr)
	if err != nil {
		return nil, err
	}

	return &lr, nil
}

// HasPermission returns a boolean representing whether the provided user has given permissions or not.
func HasPermission(s *discordgo.State, uID string, cID string, perm int64) bool {
	p, err := s.UserChannelPermissions(uID, cID)
	if err != nil {
		return false
	}

	return p&perm == perm
}
