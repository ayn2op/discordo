package chat

import (
	"fmt"
	"log/slog"
	"slices"
	"sync"

	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/discordo/internal/ui"
	"github.com/ayn2op/discordo/internal/voice"
	"github.com/ayn2op/tview"
	"github.com/ayn2op/tview/help"
	"github.com/ayn2op/tview/keybind"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/gdamore/tcell/v3"
)

type voicePanel struct {
	*tview.TextView
	cfg      *config.Config
	chatView *View

	mu           sync.RWMutex
	participants map[discord.UserID]*voiceParticipant
	order        []discord.UserID
	channelName  string
	lastError    string
	pending      bool
}

type voiceParticipant struct {
	name     string
	speaking bool
	muted    bool
	deafened bool
}

var _ help.KeyMap = (*voicePanel)(nil)

func newVoicePanel(cfg *config.Config, chatView *View) *voicePanel {
	vp := &voicePanel{
		TextView:     tview.NewTextView(),
		cfg:          cfg,
		chatView:     chatView,
		participants: make(map[discord.UserID]*voiceParticipant),
	}

	vp.Box = ui.ConfigureBox(vp.Box, &cfg.Theme)
	vp.SetTitle("Voice")
	vp.TextView.SetWrap(false)

	return vp
}

func (vp *voicePanel) ShortHelp() []keybind.Keybind {
	cfg := vp.cfg.Keybinds.Voice
	return []keybind.Keybind{cfg.ToggleMute.Keybind, cfg.ToggleDeafen.Keybind, cfg.LeaveVoice.Keybind}
}

func (vp *voicePanel) FullHelp() [][]keybind.Keybind {
	cfg := vp.cfg.Keybinds.Voice
	return [][]keybind.Keybind{
		{cfg.ToggleMute.Keybind, cfg.ToggleDeafen.Keybind, cfg.LeaveVoice.Keybind},
	}
}

func (vp *voicePanel) setChannelName(name string) {
	vp.mu.Lock()
	vp.channelName = name
	vp.mu.Unlock()
}

func (vp *voicePanel) setError(message string) {
	vp.mu.Lock()
	vp.lastError = message
	vp.mu.Unlock()
}

func (vp *voicePanel) setPending(pending bool) {
	vp.mu.Lock()
	vp.pending = pending
	vp.mu.Unlock()
}

func (vp *voicePanel) addParticipant(userID discord.UserID, name string) {
	vp.mu.Lock()
	if _, exists := vp.participants[userID]; !exists {
		vp.order = append(vp.order, userID)
	}
	vp.participants[userID] = &voiceParticipant{name: name}
	vp.mu.Unlock()
}

func (vp *voicePanel) removeParticipant(userID discord.UserID) {
	vp.mu.Lock()
	delete(vp.participants, userID)
	vp.order = slices.DeleteFunc(vp.order, func(id discord.UserID) bool { return id == userID })
	vp.mu.Unlock()
}

func (vp *voicePanel) setSpeaking(userID discord.UserID, speaking bool) {
	vp.mu.Lock()
	if p, ok := vp.participants[userID]; ok {
		p.speaking = speaking
	}
	vp.mu.Unlock()
}

func (vp *voicePanel) setParticipantMuted(userID discord.UserID, muted bool) {
	vp.mu.Lock()
	if p, ok := vp.participants[userID]; ok {
		p.muted = muted
	}
	vp.mu.Unlock()
}

func (vp *voicePanel) setParticipantDeafened(userID discord.UserID, deafened bool) {
	vp.mu.Lock()
	if p, ok := vp.participants[userID]; ok {
		p.deafened = deafened
	}
	vp.mu.Unlock()
}

func (vp *voicePanel) clear() {
	vp.mu.Lock()
	vp.participants = make(map[discord.UserID]*voiceParticipant)
	vp.order = nil
	vp.channelName = ""
	vp.lastError = ""
	vp.pending = false
	vp.mu.Unlock()
	vp.SetLines(nil)
}

func (vp *voicePanel) shouldShow() bool {
	vp.mu.RLock()
	hasError := vp.lastError != ""
	pending := vp.pending
	vp.mu.RUnlock()
	if hasError || pending {
		return true
	}
	if vp.chatView.voiceManager == nil {
		return false
	}
	return vp.chatView.voiceManager.State() != voice.VoiceDisconnected
}

func (vp *voicePanel) preferredHeight() int {
	vp.mu.RLock()
	count := len(vp.order)
	hasError := vp.lastError != ""
	pending := vp.pending
	vp.mu.RUnlock()

	height := 2
	if count > 0 {
		height += min(count, 5)
	}
	if hasError || pending {
		height++
	}
	return height
}

func (vp *voicePanel) render() {
	vp.mu.RLock()
	defer vp.mu.RUnlock()

	builder := tview.NewLineBuilder()
	if vp.channelName != "" {
		builder.Write("Channel: "+vp.channelName, tcell.StyleDefault.Bold(true))
		builder.NewLine()
	}

	firstParticipant := true
	for _, userID := range vp.order {
		p, ok := vp.participants[userID]
		if !ok {
			continue
		}
		if !firstParticipant {
			builder.NewLine()
		}
		firstParticipant = false

		style := tcell.StyleDefault
		if p.speaking {
			style = style.Bold(true)
		}
		builder.Write(p.name, style)
		if p.muted {
			builder.Write(" [M]", style.Dim(true))
		}
		if p.deafened {
			builder.Write(" [D]", style.Dim(true))
		}
	}

	// Connection status
	if vp.chatView.voiceManager != nil {
		state := vp.chatView.voiceManager.State()
		if !firstParticipant || vp.channelName != "" {
			builder.NewLine()
		}
		statusState := state.String()
		if vp.pending && state == voice.VoiceDisconnected {
			statusState = voice.VoiceConnecting.String()
		}
		status := fmt.Sprintf("Status: %s", statusState)
		dimStyle := tcell.StyleDefault.Dim(true)
		if state == voice.VoiceConnected {
			if vp.chatView.voiceManager.IsMuted() {
				status += " [M]"
			}
			if vp.chatView.voiceManager.IsDeafened() {
				status += " [D]"
			}
		}
		builder.Write(status, dimStyle)
	}

	if vp.lastError != "" {
		builder.NewLine()
		builder.Write("Error: "+vp.lastError, tcell.StyleDefault.Foreground(tcell.ColorIndianRed))
	}

	vp.SetLines(builder.Finish())
}

func (vp *voicePanel) HandleEvent(event tcell.Event) tview.Command {
	switch event := event.(type) {
	case *tview.KeyEvent:
		redraw := tview.RedrawCommand{}
		switch {
		case keybind.Matches(event, vp.cfg.Keybinds.Voice.LeaveVoice.Keybind):
			if !vp.chatView.withActiveVoiceManager(func(vm *voice.VoiceManager) {
				go func() {
					if err := vm.Leave(); err != nil {
						slog.Error("failed to leave voice channel", "err", err)
					}
					vp.chatView.app.QueueUpdateDraw(vp.chatView.updateVoiceStatus)
				}()
			}) {
				return nil
			}
			return redraw
		case keybind.Matches(event, vp.cfg.Keybinds.Voice.ToggleMute.Keybind):
			if !vp.chatView.withConnectedVoiceManager(func(vm *voice.VoiceManager) {
				if _, err := vm.ToggleMute(); err != nil {
					slog.Error("failed to toggle voice mute", "err", err)
				}
				vp.chatView.updateVoiceStatus()
			}) {
				return nil
			}
			return redraw
		case keybind.Matches(event, vp.cfg.Keybinds.Voice.ToggleDeafen.Keybind):
			if !vp.chatView.withConnectedVoiceManager(func(vm *voice.VoiceManager) {
				if _, err := vm.ToggleDeafen(); err != nil {
					slog.Error("failed to toggle voice deafen", "err", err)
				}
				vp.chatView.updateVoiceStatus()
			}) {
				return nil
			}
			return redraw
		}
		return nil
	}
	return vp.TextView.HandleEvent(event)
}
