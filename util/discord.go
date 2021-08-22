package util

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/rigormorrtiss/discordgo"
	"github.com/rivo/tview"
)

// WriteMessage parses and writes the parsed message to the provided textview.
func WriteMessage(v *tview.TextView, m *discordgo.Message, clientID string) {
	var b strings.Builder
	switch m.Type {
	case discordgo.MessageTypeDefault, discordgo.MessageTypeReply:
		parseMessage(v, &b, m, clientID)
		fmt.Fprintln(v, b.String())
	case discordgo.MessageTypeGuildMemberJoin:
		b.WriteString("[#5865F2]")
		b.WriteString(m.Author.Username)
		b.WriteString("[-]")
		b.WriteString(" joined the server")
		fmt.Fprintln(v, b.String())
	}
}

func parseMessage(v *tview.TextView, b *strings.Builder, m *discordgo.Message, clientID string) {
	// $  ╭ AUTHOR_USERNAME (BOT) MESSAGE_CONTENT*linebreak*
	parseReferencedMessage(b, m.ReferencedMessage, clientID)
	// $ AUTHOR_USERNAME (BOT)*spacee*
	parseAuthor(b, m.Author, clientID)
	// $ MESSAGE_CONTENT
	parseContent(b, m, clientID)
	// $ *space*(edited)
	parseEditedTimestamp(b, m.EditedTimestamp)
	// $ *linebreak*EMBED
	parseEmbeds(b, m.Embeds)
	// $ *linebreak*ATTACHMENT_URL
	parseAttachments(b, m.Attachments)
}

func parseContent(b *strings.Builder, m *discordgo.Message, clientID string) {
	if m.Content != "" {
		m.Content = parseMessageMentions(m.Content, m.Mentions, clientID)
		b.WriteString(m.Content)
	}
}

func parseEditedTimestamp(b *strings.Builder, t discordgo.Timestamp) {
	if t != "" {
		b.WriteString(" [::d](edited)[::-]")
	}
}

func parseMessageMentions(content string, mentions []*discordgo.User, clientID string) string {
	for _, mUser := range mentions {
		var color string
		if mUser.ID == clientID {
			color = "[#000000:#FEE75C]"
		} else {
			color = "[:#5865F2]"
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

func parseEmbeds(b *strings.Builder, embeds []*discordgo.MessageEmbed) {
	for range embeds {
		b.WriteString("\n<EMBED>")
	}
}

func parseAttachments(b *strings.Builder, attachments []*discordgo.MessageAttachment) {
	for _, a := range attachments {
		b.WriteString("\n[")
		b.WriteString(a.Filename)
		b.WriteString("]: ")
		b.WriteString(a.URL)
	}
}

func parseAuthor(b *strings.Builder, u *discordgo.User, clientID string) {
	if u.ID == clientID {
		b.WriteString("[#57F287]")
	} else {
		b.WriteString("[#ED4245]")
	}

	b.WriteString(u.Username)
	b.WriteString("[-] ")

	if u.Bot {
		b.WriteString("[#EB459E]BOT[-] ")
	}
}

func parseReferencedMessage(b *strings.Builder, rm *discordgo.Message, clientID string) {
	if rm != nil {
		b.WriteString(" ╭ ")

		if rm.Author.ID == clientID {
			b.WriteString("[#57F287::d]")
		} else {
			b.WriteString("[#ED4245::d]")
		}

		b.WriteString(rm.Author.Username)
		b.WriteString("[-] ")

		if rm.Author.Bot {
			b.WriteString("[#EB459E]BOT[-] ")
		}

		if rm.Content != "" {
			rm.Content = parseMessageMentions(rm.Content, rm.Mentions, clientID)
			b.WriteString(rm.Content)
		}
		b.WriteString("[::-]\n")
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
	var data struct {
		Code   string `json:"code"`
		Ticket string `json:"ticket"`
	}
	data.Code = code
	data.Ticket = ticket

	endpoint := discordgo.EndpointAuth + "mfa/totp"
	resp, err := s.RequestWithBucketID("POST", endpoint, data, endpoint)
	if err != nil {
		return nil, err
	}

	var lr *loginResponse
	err = json.Unmarshal(resp, &lr)
	if err != nil {
		return nil, err
	}

	return lr, nil
}
