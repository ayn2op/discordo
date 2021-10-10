package main

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/ayntgl/discordgo"
	"github.com/gen2brain/beeep"
	"github.com/rivo/tview"
)

var (
	session         *discordgo.Session
	selectedChannel *discordgo.Channel
	selectedMessage *discordgo.Message
)
var (
	boldRegex          = regexp.MustCompile(`(?m)\*\*(.*?)\*\*`)
	italicRegex        = regexp.MustCompile(`(?m)\*(.*?)\*`)
	underlineRegex     = regexp.MustCompile(`(?m)__(.*?)__`)
	strikeThroughRegex = regexp.MustCompile(`(?m)~~(.*?)~~`)
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
	dmNode := tview.NewTreeNode("Direct Messages").
		Collapse()

	n := channelsTree.GetRoot()
	n.AddChild(dmNode)

	sort.Slice(r.PrivateChannels, func(i, j int) bool {
		return r.PrivateChannels[i].LastMessageID > r.PrivateChannels[j].LastMessageID
	})

	for _, c := range r.PrivateChannels {
		var tag string
		if isUnread(c) {
			tag = "[::b]"
		} else {
			tag = "[::d]"
		}

		cn := tview.NewTreeNode(tag + generateChannelRepr(c) + "[::-]").
			SetReference(c.ID)
		dmNode.AddChild(cn)
	}

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

	for _, g := range r.Guilds {
		gn := tview.NewTreeNode(g.Name).Collapse()
		n.AddChild(gn)

		cs := g.Channels
		sort.Slice(cs, func(i, j int) bool {
			return cs[i].Position < cs[j].Position
		})

		// Top-level channels
		createTopLevelChannelsTreeNodes(gn, cs)
		// Category channels
		createCategoryChannelsTreeNodes(gn, cs)
		// Second-level channels
		createSecondLevelChannelsTreeNodes(cs)
	}

	channelsTree.SetCurrentNode(n)
}

func isUnread(c *discordgo.Channel) bool {
	if c.LastMessageID == "" {
		return false
	}

	for _, rs := range session.State.ReadState {
		if c.ID == rs.ID {
			return c.LastMessageID != rs.LastMessageID
		}
	}

	return false
}

func onSessionMessageCreate(_ *discordgo.Session, m *discordgo.MessageCreate) {
	c, err := session.State.Channel(m.ChannelID)
	if err != nil {
		return
	}

	if selectedChannel == nil || selectedChannel.ID != m.ChannelID {
		if conf.Notifications {
			for _, u := range m.Mentions {
				if u.ID == session.State.User.ID {
					g, err := session.State.Guild(m.GuildID)
					if err != nil {
						return
					}

					go beeep.Alert(fmt.Sprintf("%s (#%s)", g.Name, c.Name), m.ContentWithMentionsReplaced(), "")
					return
				}
			}
		}

		cn := getTreeNodeByReference(c.ID)
		if cn == nil {
			return
		}
		cn.SetText("[::b]" + generateChannelRepr(c) + "[::-]")
		app.Draw()
	} else {
		selectedChannel.Messages = append(selectedChannel.Messages, m.Message)
		renderMessage(m.Message)
	}
}

type loginResponse struct {
	MFA    bool   `json:"mfa"`
	SMS    bool   `json:"sms"`
	Ticket string `json:"ticket"`
	Token  string `json:"token"`
}

func login(email, password string) (*loginResponse, error) {
	data := struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}{email, password}
	resp, err := session.RequestWithBucketID(
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

func totp(code, ticket string) (*loginResponse, error) {
	data := struct {
		Code   string `json:"code"`
		Ticket string `json:"ticket"`
	}{code, ticket}
	e := discordgo.EndpointAuth + "mfa/totp"
	resp, err := session.RequestWithBucketID("POST", e, data, e)
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

func renderMessage(m *discordgo.Message) {
	var b strings.Builder

	switch m.Type {
	case discordgo.MessageTypeDefault, discordgo.MessageTypeReply:
		// Define a new region and assign message ID as the region ID.
		// Learn more:
		// https://pkg.go.dev/github.com/rivo/tview#hdr-Regions_and_Highlights
		b.WriteString("[\"")
		b.WriteString(m.ID)
		b.WriteString("\"]")
		// Render the message associated with crosspost, channel follow add,
		// pin, or a reply.
		if rm := m.ReferencedMessage; rm != nil {
			b.WriteString(" ╭ ")
			b.WriteString("[::d]")
			parseAuthor(&b, rm.Author)

			if rm.Content != "" {
				rm.Content = parseMentions(rm.Content, rm.Mentions)
				b.WriteString(parseMarkdown(rm.Content))
			}

			b.WriteString("[::-]")
			b.WriteByte('\n')
		}

		// Render the author of the message.
		parseAuthor(&b, m.Author)
		// If the message content is not empty, parse the message mentions
		// (users mentioned in the message) and render the message content.
		if m.Content != "" {
			m.Content = parseMentions(m.Content, m.Mentions)
			b.WriteString(parseMarkdown(m.Content))
		}
		// If the edited timestamp of the message is not empty; it implies that
		// the message has been edited, hence render the message with edited
		// label for distinction
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
		// Tags with no region ID ([""]) do not start new regions. They can
		// therefore be used to mark the end of a region.
		b.WriteString("[\"\"]")
		b.WriteByte('\n')
	case discordgo.MessageTypeGuildMemberJoin:
		b.WriteString("[#5865F2]")
		b.WriteString(m.Author.Username)
		b.WriteString("[-] joined the server")
		b.WriteByte('\n')
	}

	if str := b.String(); str != "" {
		b := make([]byte, len(str)+1)
		copy(b, str)

		messagesView.Write(b)
	}
}

func parseMentions(content string, mentions []*discordgo.User) string {
	for _, mUser := range mentions {
		var color string
		if mUser.ID == session.State.User.ID {
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

func parseAuthor(b *strings.Builder, u *discordgo.User) {
	if u.ID == session.State.User.ID {
		b.WriteString("[#57F287]")
	} else {
		b.WriteString("[#ED4245]")
	}

	b.WriteString(u.Username)
	b.WriteString("[-] ")
	// If the message author is a bot account, render the message with bot label
	// for distinction.
	if u.Bot {
		b.WriteString("[#EB459E]BOT[-] ")
	}
}

func parseMarkdown(md string) string {
	var res string
	res = boldRegex.ReplaceAllString(md, "[::b]$1[::-]")
	res = italicRegex.ReplaceAllString(res, "[::i]$1[::-]")
	res = underlineRegex.ReplaceAllString(res, "[::u]$1[::-]")
	res = strikeThroughRegex.ReplaceAllString(res, "[::s]$1[::-]")

	return res
}
