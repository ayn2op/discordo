package cmd

import (
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"regexp"
	"time"
	"slices"
	"fmt"

	"github.com/sahilm/fuzzy"
	"github.com/atotto/clipboard"
	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/discordo/internal/consts"
	"github.com/ayn2op/discordo/internal/ui"
	"github.com/ayn2op/discordo/internal/cache"
	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/state"
	"github.com/diamondburned/arikawa/v3/utils/json/option"
	"github.com/diamondburned/ningen/v3/discordmd"
	"github.com/gdamore/tcell/v2"
	"github.com/yuin/goldmark/ast"
	"github.com/rivo/tview"
)

const tmpFilePattern = consts.Name + "_*.md"
var mentionRegex *regexp.Regexp = regexp.MustCompile("@[a-zA-Z0-9._]+")

type messageInput struct {
	*tview.TextArea
	cfg             *config.Config
	cache           *cache.Cache
	candidates      *tview.List
	replyMessageID  discord.MessageID
	isTabCompleting bool
	lastSearch      time.Time
}

type memberList []discord.Member

func newMessageInput(cfg *config.Config) *messageInput {
	mi := &messageInput{
		TextArea:   tview.NewTextArea(),
		cfg:        cfg,
		cache:      cache.NewCache(),
		candidates: tview.NewList(),
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

	mi.candidates.Box = ui.NewConfiguredBox(mi.candidates.Box, &mi.cfg.Theme)
	mi.candidates.SetTitle("Mention")
	mi.candidates.
		ShowSecondaryText(false).
		SetSelectedStyle(tcell.StyleDefault.
			Background(tcell.ColorWhite).
			Foreground(tcell.ColorBlack)).
		SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
			if event.Name() == mi.cfg.Keys.MessageInput.Cancel {
				app.SetFocus(mi)
				return nil
			}
			return event
		})
	mi.candidates.SetRect(0, 0, 3, 3)
	return mi
}

func (mi *messageInput) reset() {
	mi.replyMessageID = 0
	mi.SetTitle("")
	mi.SetText("", true)
}

func (mi *messageInput) onInputCapture(event *tcell.EventKey) *tcell.EventKey {
	switch event.Name() {
	case mi.cfg.Keys.MessageInput.Send:
		if mi.isTabCompleting {
			mi.tabComplete(false)
		}
		mi.send()
		return nil
	case mi.cfg.Keys.MessageInput.Editor:
		mi.stopTabCompletion()
		mi.editor()
		return nil
	case mi.cfg.Keys.MessageInput.Cancel:
		if mi.isTabCompleting {
			mi.stopTabCompletion()
		} else {
			mi.reset()
		}
		return nil
	case mi.cfg.Keys.MessageInput.TabComplete:
		mi.isTabCompleting = true
		go app.QueueUpdateDraw(func(){ mi.tabComplete(false) })
		return nil
	case "Rune[@]":
		mi.isTabCompleting = true
		go app.QueueUpdateDraw(func(){ mi.tabComplete(true) })
		return event
	}

	if mi.isTabCompleting {
		k := event.Key()
		if (k == tcell.KeyRune && isValidUserRune(event.Rune())) ||
		    k == tcell.KeyBackspace || k == tcell.KeyBackspace2 {
			go app.QueueUpdateDraw(func(){ mi.tabComplete(true) })
			return event
		}
		c := mi.candidates.GetItemCount()
		n := event.Name()
		switch n {
		case "Down", "Up":
			if c > 1 {
				if n == "Down" {
					c = 1
				}
				mi.candidates.SetCurrentItem(c)
			}
			app.SetFocus(mi.candidates)
			return nil
		}
		mi.stopTabCompletion()
	}

	return event
}

func (mi *messageInput) send() {
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
	return mentionRegex.ReplaceAllFunc(src, func(in []byte) (out []byte) {
		out = in
		name := strings.ToLower(string(in[1:]))
		discordState.MemberStore.Each(app.guildsTree.selectedGuildID, func (m *discord.Member) bool {
			if strings.ToLower(m.User.Username) == name {
				out = []byte("<@" + m.User.ID.String() + ">")
				return true
			}
			return false
		})
		return
	})
}

func (mi *messageInput) tabComplete(isAuto bool) {
	gID := app.guildsTree.selectedGuildID
	cID := app.guildsTree.selectedChannelID
	// This is stupid, why can't tview just give us the real textline of
	// the cursor when wordwrapping??
	row, col, _, _ := mi.GetCursor()
	_, _, w, _ := mi.GetInnerRect()
	lines := strings.SplitN(mi.GetText(), "\n", row+2)
	oldlineslen := len(lines)
	for i := range lines {
		lines = append(lines, tview.WordWrap(lines[i], w)...)
	}
	lines = slices.Delete(lines, 0, oldlineslen)
	lines = slices.Delete(lines, row+1, len(lines))
	left := strings.TrimRightFunc(lines[row][:col], isValidUserRune)
	if len(left) == 0 || left[len(left)-1] != '@' {
		mi.stopTabCompletion()
		return
	}
	name := strings.TrimPrefix(lines[row][:col], left)
	pos := len(left) + len(strings.Join(lines[:row], "\n")) - (len(lines)-oldlineslen)
	if row == 0 {
		pos--
	}
	posEnd := pos + len(name)+1

	if !isAuto && mi.candidates.GetItemCount() != 0 {
		_, name = mi.candidates.GetItemText(0)
		mi.Replace(pos, posEnd, "@" + name + " ")
		mi.stopTabCompletion()
		return
	}

	// Special case, show recent messages' authors
	if name == "" {
		msgs, err := discordState.MessageStore.Messages(cID)
		if err != nil {
			return
		}
		shown := make(map[string]bool)
		mi.candidates.Clear()
		for _, m := range msgs {
			if shown[m.Author.Username] {
				continue
			}
			shown[m.Author.Username] = true
			if mem, ok := discordState.Cabinet.Member(gID, m.Author.ID); ok == nil {
				if mi.addCandidate(gID, mem) {
					break
				}
			}
		}
	} else {
		mi.searchMember(gID, name)
		mi.candidates.Clear()
		mems, _ := discordState.Cabinet.Members(gID)
		res := fuzzy.FindFrom(name, memberList(mems))
		if len(res) > mi.cfg.Theme.Candidates.ListLimit {
			res = res[:mi.cfg.Theme.Candidates.ListLimit]
		}
		for _, r := range res {
			if mi.addCandidate(gID, &mems[r.Index]) {
				break
			}
		}
	}

	if mi.candidates.GetItemCount() == 0 {
		mi.stopTabCompletion()
		return
	}
	mi.choose(min(col, len(left)), pos, posEnd)
}

func (m memberList) String(i int) string { return m[i].Nick + m[i].User.DisplayName + m[i].User.Tag() }
func (m memberList) Len() int { return len(m) }

func (mi *messageInput) searchMember(gID discord.GuildID, name string) {
	key := gID.String() + " " + name
	if mi.cache.Exists(key) {
		return
	}
	// If searching for "ab" returns less than SearchLimit,
	// then "abc" would not return anything new because we already searched
	// everything starting with "ab". This will still be true even if a new
	// member joins because arikawa loads new members into the state.
	if k := key[:len(key)-1]; mi.cache.Exists(k) {
		if c := mi.cache.Get(k);
		   c < discordState.MemberState.SearchLimit {
			mi.cache.Create(key, c)
			return
		}
	}
	// Rate limit on our side because we can't distinguish between a
	// successful search and SearchMember not doing anything becuase of its
	// internal rate limit that we can't detect
	if mi.lastSearch.Add(discordState.MemberState.SearchFrequency).After(time.Now()) {
		return
	}
	mi.lastSearch = time.Now()
	app.messagesText.waitForChunkEvent()
	app.messagesText.setFetchingChunk(true, 0)
	discordState.MemberState.SearchMember(gID, name)
	mi.cache.Create(key, app.messagesText.waitForChunkEvent())
}


func isValidUserRune(x rune) bool {
	return (x >= 'a' && x <= 'z') ||
	       (x >= 'A' && x <= 'Z') ||
	       (x >= '0' && x <= '9') ||
	        x == '_' || x == '.'
}

func (mi *messageInput) choose(col, pos, posEnd int) {
	l := mi.candidates
	x, _, _, _ := mi.GetInnerRect()
	_, y, w, _ := mi.GetRect()
	_, _, _, maxH := app.messagesText.GetRect()
	h := min(l.GetItemCount(), maxH-5)
	// Don't add the top padding because it is automatically added by tview
	h += mi.cfg.Theme.Border.Padding[1]
	if col - 1 > w - 22 {
		l.SetRect(x + w - 22, y - h - 2, 20, h + 2)
	} else {
		l.SetRect(x + col - 1, y - h - 2, 20, h + 2)
	}
	l.SetSelectedFunc(func (_ int, _, username string, _ rune) {
		mi.Replace(pos, posEnd, "@" + username[2:] + " ")
		mi.stopTabCompletion()
	})
	app.pages.ShowPage("candidates")
	app.SetFocus(mi)
}

func (mi *messageInput) addCandidate(gID discord.GuildID, m *discord.Member) bool {
	var text string
	dname := m.User.DisplayName
	// this is WAY faster than the old discordState.MemberColor
	if !mi.cfg.Theme.Candidates.ShowUsernameColors {
		goto NO_COLOR
	}
	if c, ok := state.MemberColor(m, func(id discord.RoleID) *discord.Role {
		r, _ := discordState.Cabinet.Role(gID, id)
		return r
	}); ok {
		if dname != "" {
			text = fmt.Sprintf("[%s]%s[-] (%s)", c.String(), tview.Escape(dname), m.User.Username)
		} else {
			text = fmt.Sprintf("[%s]%s[-]", c.String(), m.User.Username)
		}
		goto END
	}
	NO_COLOR:
	if dname != "" {
		text = fmt.Sprintf("%s (%s)", tview.Escape(dname), m.User.Username)
	} else {
		text = m.User.Username
	}
	END:
	mi.candidates.AddItem(text, m.User.Username, 0, nil)
	return mi.candidates.GetItemCount() > mi.cfg.Theme.Candidates.ListLimit
}

func (mi *messageInput) stopTabCompletion() {
	app.pages.HidePage("candidates")
	mi.candidates.Clear()
	app.SetFocus(mi)
	mi.isTabCompleting = false
}

func (mi *messageInput) editor() {
	e := mi.cfg.Editor
	if e == "default" {
		e = os.Getenv("EDITOR")
	}

	file, err := os.CreateTemp("", tmpFilePattern)
	if err != nil {
		slog.Error("failed to create tmp file", "err", err)
		return
	}
	defer file.Close()
	defer os.Remove(file.Name())

	_, _ = file.WriteString(mi.GetText())

	cmd := exec.Command(e, file.Name())
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	app.Suspend(func() {
		err := cmd.Run()
		if err != nil {
			slog.Error("failed to run command", "args", cmd.Args, "err", err)
			return
		}
	})

	msg, err := os.ReadFile(file.Name())
	if err != nil {
		slog.Error("failed to read tmp file", "name", file.Name(), "err", err)
		return
	}

	mi.SetText(strings.TrimSpace(string(msg)), true)
}
