package cmd

import (
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"regexp"
	"time"

	"github.com/atotto/clipboard"
	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/discordo/internal/consts"
	"github.com/ayn2op/discordo/internal/ui"
	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/utils/json/option"
	"github.com/diamondburned/ningen/v3/discordmd"
	"github.com/gdamore/tcell/v2"
	"github.com/yuin/goldmark/ast"
	"github.com/rivo/tview"
)

const tmpFilePattern = consts.Name + "_*.md"

type MessageInput struct {
	*tview.TextArea
	cfg            *config.Config
	app            *tview.Application
	replyMessageID discord.MessageID
	lastSearch     time.Time
}

func newMessageInput(app *tview.Application, cfg *config.Config) *MessageInput {
	mi := &MessageInput{
		TextArea: tview.NewTextArea(),
		cfg:      cfg,
		app:      app,
	}

	mi.Box = ui.NewConfiguredBox(mi.Box, &cfg.Theme)

	mi.
		SetTextStyle(tcell.StyleDefault.Background(tcell.GetColor(cfg.Theme.BackgroundColor))).
		SetClipboard(func(s string) {
			_ = clipboard.WriteAll(s)
		}, func() string {
			text, _ := clipboard.ReadAll()
			return text
		}).
		SetInputCapture(mi.onInputCapture)

	return mi
}

func (mi *MessageInput) reset() {
	mi.replyMessageID = 0
	mi.SetTitle("")
	mi.SetText("", true)
}

func (mi *MessageInput) onInputCapture(event *tcell.EventKey) *tcell.EventKey {
	switch event.Name() {
	case mi.cfg.Keys.MessageInput.Send:
		mi.send()
		return nil
	case mi.cfg.Keys.MessageInput.Editor:
		mi.editor()
		return nil
	case mi.cfg.Keys.MessageInput.Cancel:
		mi.reset()
		return nil
	case mi.cfg.Keys.MessageInput.TabComplete:
		mi.tabComplete()
		return nil
	}

	return event
}

func (mi *MessageInput) send() {
	if !app.guildsTree.selectedChannelID.IsValid() {
		return
	}

	text := strings.TrimSpace(mi.GetText())
	if text == "" {
		return
	}

	// Process mentions (there's no shortcut, just parse the entire message
	// as markdown and then re-emit the content with proper mentions)
	data := api.SendMessageData{
		Content: processText([]byte(text)),
	}
	if mi.replyMessageID != 0 {
		data.Reference = &discord.MessageReference{MessageID: mi.replyMessageID}
		data.AllowedMentions = &api.AllowedMentions{RepliedUser: option.False}

		if strings.HasPrefix(mi.GetTitle(), "[@]") {
			data.AllowedMentions.RepliedUser = option.True
		}
	}

	go func() {
		if _, err := discordState.SendMessageComplex(app.guildsTree.selectedChannelID, data); err != nil {
			slog.Error("failed to send message in channel", "channel_id", app.guildsTree.selectedChannelID, "err", err)
		}
	}()

	mi.replyMessageID = 0
	mi.reset()

	app.messagesText.Highlight()
	app.messagesText.ScrollToEnd()
}

func processText(src []byte) string {
	canMention := true
	n := discordmd.Parse(src)
	var res strings.Builder
	ast.Walk(n, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		switch n := n.(type) {
		case *ast.Heading:
			if entering {
				for range n.Level {
					res.WriteByte('#')
				}
				res.WriteByte(' ')
			} else {
				res.WriteByte('\n')
			}
		case *ast.Text:
			if entering {
				break
			}
			if canMention {
				res.Write(expandMentions(n.Value(src)))
			} else {
				res.Write(n.Value(src))
			}
			switch {
			case n.HardLineBreak(): res.WriteByte('\n')
			case n.SoftLineBreak(): res.WriteByte(' ')
			}

		case *ast.FencedCodeBlock:
			canMention = !entering
			if entering {
				break
			}
			res.WriteString("```")
			if l := n.Language(src); l != nil {
				res.Write(l)
			}
			res.WriteByte('\n')
			for i := range n.Lines().Len() {
				line := n.Lines().At(i)
				res.Write(line.Value(src))
			}
			res.WriteString("```\n")

		case *ast.AutoLink:
			if entering {
				break
			}
			res.Write(n.URL(src))

		case *ast.Link:
			if entering {
				res.WriteByte('[')
				res.Write(n.Title)
				res.WriteByte(']')
				res.WriteByte('(')
				res.Write(n.Destination)
				res.WriteByte(')')
			}

		case *discordmd.Inline:
			switch n.Attr {
			case discordmd.AttrBold:          res.WriteString("**")
			case discordmd.AttrItalics:       res.WriteByte('*')
			case discordmd.AttrUnderline:     res.WriteString("__")
			case discordmd.AttrStrikethrough: res.WriteString("~~")
			case discordmd.AttrSpoiler:       res.WriteString("||")
			case discordmd.AttrMonospace:
				canMention = !entering
				res.WriteByte('`')
			}

		case *discordmd.Emoji:
			if entering {
				res.WriteByte(':')
				res.WriteString(n.Name)
				res.WriteByte(':')
			}
		}
		return ast.WalkContinue, nil
	})
	return res.String()
}

func expandMentions(src []byte) []byte {
	rx := regexp.MustCompile("@([a-zA-Z0-9._]+)")
	return rx.ReplaceAllFunc(src, func(in []byte) []byte {
		if id := findUser(string(in[1:]));
		   id != discord.NullUserID {
			return []byte("@" + id.String())
		} else {
			return in
		}
	})
}

func findUser(name string) discord.UserID {
	gID := app.guildsTree.selectedGuildID
	id := discord.NullUserID
	discordState.MemberStore.Each(gID, func (m *discord.Member) bool {
		if m.User.Username == name {
			id = m.User.ID
			return true
		}
		return false
	})
	if id != discord.NullUserID {
		return id
	}
	discordState.MemberStore.Each(gID, func (m *discord.Member) bool {
		if m.User.DisplayName == name {
			id = m.User.ID
			return true
		}
		return false
	})
	return id
}

func (mi *MessageInput) tabComplete() {
	row, col, _, _ := mi.GetCursor()
	lines := strings.Split(mi.GetText(), "\n")
	left := strings.TrimRightFunc(lines[row][:col], func (x rune) bool {
		return (x >= 'a' && x <= 'z') ||
		       (x >= 'A' && x <= 'Z') ||
		       (x >= '0' && x <= '9')
	})
	if !strings.HasSuffix(left, "@") {
		return
	}
	name := strings.TrimPrefix(lines[row][:col], left)

	// Special case, show recent messages' authors
	if name == "" {
		msgs, err := discordState.MessageStore.Messages(app.guildsTree.selectedChannelID)
		if err != nil {
			app.messagesText.displayInternalMsg(true, "Error: %s", err.Error())
			return
		}
		shown := make(map[string]bool)
		app.messagesText.clearInternals = true
		app.messagesText.displayInternalMsg(false, "Possible mentions:")
		i := 0
		for _, m := range msgs {
			if shown[m.Author.Username] {
				continue
			}
			if i > app.cfg.Theme.MessagesText.CandidateListLimit {
				app.messagesText.displayInternalMsg(false, " ...")
				return
			}
			i++
			shown[m.Author.Username] = true
			listCandidate(app.guildsTree.selectedGuildID, m.Author)
		}
		return
	}

	if mi.lastSearch.Add(1 * time.Second).After(time.Now()) {
		app.messagesText.displayInternalMsg(false, "Slow down...")
		return
	}
	mi.lastSearch = time.Now()

	app.messagesText.setFetchingChunk(true)
	discordState.MemberState.SearchMember(app.guildsTree.selectedGuildID, name)
	app.messagesText.waitForChunkEvent()

	var candidates []discord.User
	var exact string
	discordState.MemberStore.Each(app.guildsTree.selectedGuildID, func (m *discord.Member) bool {
		if m.User.Username == name {
			exact = m.User.Username
			return true
		}
		if strings.HasPrefix(m.User.Username, name) {
			candidates = append(candidates, m.User)
		}
		return false
	})

	if exact == "" && len(candidates) == 1 {
		exact = candidates[0].Username
	}
	if exact != "" {
		loc := len(left) + len(strings.Join(lines[:row], "\n"))
		if row == 0 {
			loc--
		}
		mi.Replace(loc, loc + len(name)+1, "@" + exact)
		return
	}
	if len(candidates) == 0 {
		return
	}

	app.messagesText.clearInternals = true
	app.messagesText.displayInternalMsg(false, "Possible mentions:")
	for i, c := range candidates {
		if i > app.cfg.Theme.MessagesText.CandidateListLimit {
			app.messagesText.displayInternalMsg(false, " ...")
			return
		}
		listCandidate(app.guildsTree.selectedGuildID, c)
	}
}

func listCandidate(gID discord.GuildID, user discord.User) {
	if c, ok := discordState.MemberColor(gID, user.ID); ok {
		app.messagesText.displayInternalMsg(false, "* %15s ([%s]%s[-])", user.Username, c.String(), user.DisplayOrUsername())
		return
	}
	app.messagesText.displayInternalMsg(false, "* %15s (%s)", user.Username, user.DisplayOrUsername())
}

func (mi *MessageInput) editor() {
	e := mi.cfg.Editor
	if e == "default" {
		e = os.Getenv("EDITOR")
	}

	f, err := os.CreateTemp("", tmpFilePattern)
	if err != nil {
		slog.Error("failed to create tmp file", "err", err)
		return
	}
	_, _ = f.WriteString(mi.GetText())
	f.Close()

	defer os.Remove(f.Name())

	cmd := exec.Command(e, f.Name())
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	mi.app.Suspend(func() {
		err := cmd.Run()
		if err != nil {
			slog.Error("failed to run command", "args", cmd.Args, "err", err)
			return
		}
	})

	msg, err := os.ReadFile(f.Name())
	if err != nil {
		slog.Error("failed to read tmp file", "name", f.Name(), "err", err)
		return
	}

	mi.SetText(strings.TrimSpace(string(msg)), true)
}
