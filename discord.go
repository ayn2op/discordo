package main

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/ayntgl/discordgo"
	"github.com/ayntgl/discordo/util"
	"github.com/gen2brain/beeep"
	"github.com/rivo/tview"
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

	sort.Slice(r.Guilds, func(a, b int) bool {
		found := false
		for _, gID := range session.State.Settings.GuildPositions {
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
		gn := tview.NewTreeNode(g.Name).
			SetReference(g.ID).
			Collapse()
		n.AddChild(gn)
	}

	channelsTree.SetCurrentNode(n)
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

		cn := util.GetTreeNodeByReference(channelsTree, c.ID)
		if cn == nil {
			return
		}
		cn.SetText("[::b]" + util.ChannelToString(c) + "[::-]")
		app.Draw()
	} else {
		selectedChannel.Messages = append(selectedChannel.Messages, m.Message)
		messagesView.Write(buildMessage(m.Message))
		// Scroll to the end of the text after the message has been written to the TextView.
		messagesView.ScrollToEnd()
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

func buildMessage(m *discordgo.Message) []byte {
	var b strings.Builder

	switch m.Type {
	case discordgo.MessageTypeDefault, discordgo.MessageTypeReply:
		// Define a new region and assign message ID as the region ID.
		// Learn more:
		// https://pkg.go.dev/github.com/rivo/tview#hdr-Regions_and_Highlights
		b.WriteString("[\"")
		b.WriteString(m.ID)
		b.WriteString("\"]")
		// Build the message associated with crosspost, channel follow add, pin, or a reply.
		buildReferencedMessage(&b, m.ReferencedMessage)
		// Build the author of this message.
		buildAuthor(&b, m.Author)
		// Build the contents of the message.
		buildContent(&b, m)

		if m.EditedTimestamp != "" {
			b.WriteString(" [::d](edited)[::-]")
		}
		// Build the embeds associated with the message.
		buildEmbeds(&b, m.Embeds)
		// Build the message attachments (attached files to the message).
		buildAttachments(&b, m.Attachments)
		// Tags with no region ID ([""]) do not start new regions. They can
		// therefore be used to mark the end of a region.
		b.WriteString("[\"\"]")

		b.WriteByte('\n')
	case discordgo.MessageTypeGuildMemberJoin:
		b.WriteString("[#5865F2]")
		b.WriteString(m.Author.Username)
		b.WriteString("[-] joined the server.")

		b.WriteByte('\n')
	case discordgo.MessageTypeCall:
		b.WriteString("[#5865F2]")
		b.WriteString(m.Author.Username)
		b.WriteString("[-] started a call.")

		b.WriteByte('\n')
	case discordgo.MessageTypeChannelPinnedMessage:
		b.WriteString("[#5865F2]")
		b.WriteString(m.Author.Username)
		b.WriteString("[-] pinned a message.")

		b.WriteByte('\n')
	}

	if str := b.String(); str != "" {
		b := make([]byte, len(str)+1)
		copy(b, str)

		return b
	}

	return nil
}

func buildReferencedMessage(b *strings.Builder, rm *discordgo.Message) {
	if rm != nil {
		b.WriteString(" ╭ ")
		b.WriteString("[::d]")
		buildAuthor(b, rm.Author)

		if rm.Content != "" {
			rm.Content = buildMentions(rm.Content, rm.Mentions)
			b.WriteString(parseMarkdown(rm.Content))
		}

		b.WriteString("[::-]")
		b.WriteByte('\n')
	}
}

func buildContent(b *strings.Builder, m *discordgo.Message) {
	if m.Content != "" {
		m.Content = buildMentions(m.Content, m.Mentions)
		b.WriteString(parseMarkdown(m.Content))
	}
}

func buildEmbeds(b *strings.Builder, es []*discordgo.MessageEmbed) {
	for _, e := range es {
		if e.Type != discordgo.EmbedTypeRich {
			continue
		}

		var embedBuilder strings.Builder
		var hasHeading bool
		prefix := fmt.Sprintf("[#%06X]▐[-] ", e.Color)

		b.WriteByte('\n')
		embedBuilder.WriteString(prefix)

		if e.Author != nil {
			hasHeading = true
			embedBuilder.WriteString("[::u]")
			embedBuilder.WriteString(e.Author.Name)
			embedBuilder.WriteString("[::-]")
		}

		if e.Title != "" {
			hasHeading = true
			embedBuilder.WriteString("[::b]")
			embedBuilder.WriteString(e.Title)
			embedBuilder.WriteString("[::-]")
		}

		if e.Description != "" {
			if hasHeading {
				embedBuilder.WriteString("\n\n")
			}

			embedBuilder.WriteString(parseMarkdown(e.Description))
		}

		if len(e.Fields) != 0 {
			if hasHeading || e.Description != "" {
				embedBuilder.WriteString("\n\n")
			}

			for i, ef := range e.Fields {
				embedBuilder.WriteString("[::b]")
				embedBuilder.WriteString(ef.Name)
				embedBuilder.WriteString("[::-]")
				embedBuilder.WriteByte('\n')
				embedBuilder.WriteString(parseMarkdown(ef.Value))

				if i != len(e.Fields)-1 {
					embedBuilder.WriteString("\n\n")
				}
			}
		}

		if e.Footer != nil {
			if hasHeading {
				embedBuilder.WriteString("\n\n")
			}

			embedBuilder.WriteString(e.Footer.Text)
		}

		b.WriteString(strings.Replace(embedBuilder.String(), "\n", "\n"+prefix, -1))
	}
}

func buildAttachments(b *strings.Builder, as []*discordgo.MessageAttachment) {
	for _, a := range as {
		b.WriteByte('\n')
		b.WriteByte('[')
		b.WriteString(a.Filename)
		b.WriteString("]: ")
		b.WriteString(a.URL)
	}
}

func buildMentions(content string, mentions []*discordgo.User) string {
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

func buildAuthor(b *strings.Builder, u *discordgo.User) {
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
