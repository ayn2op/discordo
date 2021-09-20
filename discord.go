package main

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/ayntgl/discordgo"
	"github.com/gen2brain/beeep"
	"github.com/rivo/tview"
)

func newSession() *discordgo.Session {
	s, err := discordgo.New()
	if err != nil {
		panic(err)
	}

	s.UserAgent = conf.UserAgent
	s.Identify.Compress = false
	s.Identify.Intents = 0
	s.Identify.LargeThreshold = 0
	s.Identify.Properties.Device = ""
	s.Identify.Properties.Browser = "Chrome"
	s.Identify.Properties.OS = "Linux"

	s.AddHandlerOnce(onSessionReady)
	s.AddHandler(onSessionMessageCreate)

	return s
}

func onSessionReady(_ *discordgo.Session, r *discordgo.Ready) {
	sort.Slice(r.Guilds, func(a, b int) bool {
		found := false
		for _, gID := range r.Settings.GuildPositions {
			if found {
				if gID == r.Guilds[b].ID {
					return true
				}
			} else {
				if gID == r.Guilds[a].ID {
					found = true
				}
			}
		}

		return false
	})

	n := guildsTreeView.GetRoot()
	for _, g := range r.Guilds {
		gn := tview.NewTreeNode(g.Name).
			SetReference(g.ID)
		n.AddChild(gn)
	}

	guildsTreeView.SetCurrentNode(n)
}

func onSessionMessageCreate(_ *discordgo.Session, m *discordgo.MessageCreate) {
	if selectedChannel == nil {
		selectedChannel = &discordgo.Channel{ID: ""}
	}

	if selectedChannel.ID != m.ChannelID {
		if conf.Notifications {
			for _, u := range m.Mentions {
				if u.ID == session.State.User.ID {
					g, err := session.State.Guild(m.GuildID)
					if err != nil {
						return
					}
					c, err := session.State.Channel(m.ChannelID)
					if err != nil {
						return
					}

					go beeep.Alert(fmt.Sprintf("%s (#%s)", g.Name, c.Name), m.ContentWithMentionsReplaced(), "")
					return
				}
			}
		}

		return
	}

	selectedChannel.Messages = append(selectedChannel.Messages, m.Message)
	renderMessage(m.Message)
}

type loginResponse struct {
	MFA    bool   `json:"mfa"`
	SMS    bool   `json:"sms"`
	Ticket string `json:"ticket"`
	Token  string `json:"token"`
}

func login(
	s *discordgo.Session,
	email, password string,
) (*loginResponse, error) {
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

func totp(s *discordgo.Session, code, ticket string) (*loginResponse, error) {
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
