package discord

import (
	"encoding/json"

	"github.com/ayntgl/discordgo"
)

type LoginResponse struct {
	Ticket string `json:"ticket"`
	Token  string `json:"token"`
	MFA    bool   `json:"mfa"`
	SMS    bool   `json:"sms"`
}

func Login(s *discordgo.Session, email string, password string) (*LoginResponse, error) {
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

	var lr LoginResponse
	err = json.Unmarshal(resp, &lr)
	if err != nil {
		return nil, err
	}

	return &lr, nil
}

func TOTP(s *discordgo.Session, code string, ticket string) (*LoginResponse, error) {
	data := struct {
		Code   string `json:"code"`
		Ticket string `json:"ticket"`
	}{code, ticket}
	e := discordgo.EndpointAuth + "mfa/totp"
	resp, err := s.RequestWithBucketID("POST", e, data, e)
	if err != nil {
		return nil, err
	}

	var lr LoginResponse
	err = json.Unmarshal(resp, &lr)
	if err != nil {
		return nil, err
	}

	return &lr, nil
}
