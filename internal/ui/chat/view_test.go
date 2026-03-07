package chat

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/discordo/internal/voice"
	"github.com/ayn2op/tview"
	"github.com/diamondburned/arikawa/v3/discord"
)

func testConfig(t *testing.T) *config.Config {
	t.Helper()

	cfg, err := config.Load(filepath.Join(t.TempDir(), "missing.toml"))
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	return cfg
}

func TestPrepareVoiceJoinUIKeepsSelectedChannel(t *testing.T) {
	view := NewView(tview.NewApplication(), testConfig(t), "")
	view.voiceManager = voice.NewVoiceManager(nil, voice.Config{})

	textChannel := &discord.Channel{ID: discord.ChannelID(1), Name: "general", Type: discord.GuildText}
	voiceChannel := &discord.Channel{ID: discord.ChannelID(2), Name: "Lobby", Type: discord.GuildVoice}

	view.SetSelectedChannel(textChannel)
	view.prepareVoiceJoinUI(voiceChannel)

	if got := view.SelectedChannel(); got != textChannel {
		t.Fatalf("selected channel changed to %+v", got)
	}
	if !view.voicePanel.shouldShow() {
		t.Fatal("expected voice panel to remain visible during join")
	}
	if text := view.voicePanel.GetText(); !strings.Contains(text, "Status: Connecting") {
		t.Fatalf("expected connecting status, got %q", text)
	}
}

func TestUpdateVoiceStatusClearsPendingState(t *testing.T) {
	view := NewView(tview.NewApplication(), testConfig(t), "")
	view.voiceManager = voice.NewVoiceManager(nil, voice.Config{})

	view.voicePanel.setPending(true)
	view.updateVoiceStatus()

	if view.voicePanel.shouldShow() {
		t.Fatal("expected disconnected voice panel to hide after pending state is cleared")
	}
}

func TestFocusCycleIncludesVisibleVoicePanel(t *testing.T) {
	app := tview.NewApplication()
	view := NewView(app, testConfig(t), "")
	view.messageInput.SetDisabled(false)
	view.voicePanel.setPending(true)
	view.updateVoicePanelLayout()

	app.SetFocus(view.messagesList)
	view.focusNext()
	if app.GetFocus() != view.voicePanel {
		t.Fatal("expected focusNext to move from messages list to voice panel")
	}

	view.focusNext()
	if app.GetFocus() != view.messageInput {
		t.Fatal("expected focusNext to move from voice panel to message input")
	}

	view.focusPrevious()
	if app.GetFocus() != view.voicePanel {
		t.Fatal("expected focusPrevious to move from message input to voice panel")
	}
}

func TestActiveKeyMapReturnsVoicePanelWhenFocused(t *testing.T) {
	app := tview.NewApplication()
	view := NewView(app, testConfig(t), "")
	view.voicePanel.setPending(true)
	view.updateVoicePanelLayout()
	app.SetFocus(view.voicePanel)

	if got := view.activeKeyMap(); got != view.voicePanel {
		t.Fatalf("expected active key map to be voice panel, got %T", got)
	}
}
