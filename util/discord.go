package util

import (
	"encoding/json"
	"regexp"

	"github.com/ayntgl/discordgo"
)

var (
	boldRegex          = regexp.MustCompile(`(?m)\*\*(.*?)\*\*`)
	italicRegex        = regexp.MustCompile(`(?m)\*(.*?)\*`)
	underlineRegex     = regexp.MustCompile(`(?m)__(.*?)__`)
	strikeThroughRegex = regexp.MustCompile(`(?m)~~(.*?)~~`)
)

func ParseMarkdown(md string) string {
	var res string
	res = boldRegex.ReplaceAllString(md, "[::b]$1[::-]")
	res = italicRegex.ReplaceAllString(res, "[::i]$1[::-]")
	res = underlineRegex.ReplaceAllString(res, "[::u]$1[::-]")
	res = strikeThroughRegex.ReplaceAllString(res, "[::s]$1[::-]")

	return res
}

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

type loginResponse struct {
	Ticket string `json:"ticket"`
	Token  string `json:"token"`
	MFA    bool   `json:"mfa"`
	SMS    bool   `json:"sms"`
}

func Login(s *discordgo.Session, email string, password string) (*loginResponse, error) {
	data := struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}{email, password}
	resp, err := s.RequestWithBucketID(
		"POST",
		discordgo.EndpointLogin,
		data,
		discordgo.EndpointLogin,
	)
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

func TOTP(s *discordgo.Session, code string, ticket string) (*loginResponse, error) {
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
