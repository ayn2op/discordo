package cmd

import (
	"log/slog"
	"os"
	"os/exec"
	"strings"

	"github.com/atotto/clipboard"
	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/discordo/internal/consts"
	"github.com/ayn2op/discordo/internal/ui"
	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/utils/json/option"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

const tmpFilePattern = consts.Name + "_*.md"

type messageInput struct {
	*tview.TextArea
	cfg            *config.Config
	app            *tview.Application
	replyMessageID discord.MessageID
}

func newMessageInput(app *tview.Application, cfg *config.Config) *messageInput {
	mi := &messageInput{
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

func (mi *messageInput) reset() {
	mi.replyMessageID = 0
	mi.SetTitle("")
	mi.SetText("", true)
}

func (mi *messageInput) onInputCapture(event *tcell.EventKey) *tcell.EventKey {
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

	data := api.SendMessageData{
		Content: text,
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

func (mi *messageInput) editor() {
	e := mi.cfg.Editor
	if e == "default" {
		e = os.Getenv("EDITOR")
	}

	f, err := os.CreateTemp("", tmpFilePattern)
	if err != nil {
		slog.Error("failed to create tmp file", "err", err)
		return
	}
	defer f.Close()
	defer os.Remove(f.Name())

	_, _ = f.WriteString(mi.GetText())

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
