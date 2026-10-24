package messageslist

import (
	"fmt"
	"io"
	"slices"
	"time"

	"github.com/ayn2op/arikawa/v3/discord"
	"github.com/ayn2op/discordo/internal/ui"
	"github.com/ayn2op/tview"
	"github.com/ayn2op/tview/button"
	"github.com/ayn2op/tview/layout"
	"github.com/ayn2op/tview/row"
	"github.com/ayn2op/tview/slider"
	"github.com/ayn2op/tview/text"
)

const (
	symbolPlay  = "▶"
	symbolPause = "⏸"
	// positionSeparator is between how much of a voice message was played and how long it is.
	positionSeparator = " / "
	// waveformBars is what the waveform of a voice message is drawn with, from quiet to loud.
	waveformBars = "▁▂▃▄▅▆▇█"
	// waveformWidth is how many bars a waveform is drawn with at most.
	waveformWidth = 32
	// seekStep is how far a key seeks a voice message.
	seekStep = time.Second
)

// waveformRail returns the bars the waveform of a voice message is drawn with.
func waveformRail(waveform []byte) []string {
	bars := []rune(waveformBars)

	var rail []string
	for chunk := range slices.Chunk(waveform, len(waveform)/waveformWidth+1) {
		rail = append(rail, string(bars[int(slices.Max(chunk))*len(bars)/256]))
	}

	return rail
}

func formatDuration(d time.Duration) string {
	seconds := int(d.Round(time.Second).Seconds())
	return fmt.Sprintf("%d:%02d", seconds/60, seconds%60)
}

// voiceMessageView returns the player of the voice message attachment: whether it is playing, its waveform with what was played of it, and how long it is.
func (m Model) voiceMessageView(attachment discord.Attachment) tview.Widget {
	duration := time.Duration(attachment.DurationSecs * float64(time.Second))
	symbol, label := symbolPlay, formatDuration(duration)

	status := m.voiceStatus
	loaded := status.ID == attachment.ID
	if loaded {
		label = formatDuration(status.Position) + positionSeparator + label
	}

	if loaded && status.Playing {
		symbol = symbolPause
	}

	rail := waveformRail(attachment.Waveform)
	style := m.cfg.Theme.MessagesList.AttachmentStyle.Style
	// Only the voice message that is loaded can be sought.
	var onSeek func(time.Duration) tview.Msg
	if loaded {
		onSeek = func(position time.Duration) tview.Msg { return seekMsg(position) }
	}

	waveform := slider.New(0, duration, status.Position, onSeek).
		OnPress(func(position time.Duration) tview.Msg { return seekPressMsg(position) }).
		OnRelease(seekReleaseMsg{}).
		Dragging(loaded && m.seeking).
		Width(layout.Fixed(len(rail))).
		Rail(func(cell, handle int) (string, tview.Style) {
			if loaded && cell <= handle {
				return rail[cell], style
			}

			return rail[cell], style.Dim(true)
		})

	play := button.New(text.New(symbol)).OnClick(playMsg(attachment))

	return row.New(play, waveform, text.New(label)).Spacing(1).Height(layout.Fixed(1))
}

func isVoiceMessage(message discord.Message) bool {
	return message.Flags&discord.IsVoiceMessage != 0 && len(message.Attachments) == 1
}

// playMsg asks to play the voice message whose play symbol was clicked.
type playMsg discord.Attachment

func (playMsg) messagesList() {}

// play plays the voice message that is attachment, or pauses or resumes it if it is the one playing.
func (m Model) play(attachment discord.Attachment) tview.Cmd {
	return func() tview.Msg {
		if err := m.playVoiceMessage(attachment); err != nil {
			return ui.ErrorDialog("play voice message", err)
		}

		return nil
	}
}

func (m Model) playVoiceMessage(attachment discord.Attachment) error {
	if toggled, err := m.voicePlayer.Toggle(attachment.ID); toggled || err != nil {
		return err
	}

	resp, err := m.Get(attachment.URL)
	if err != nil {
		return err
	}
	defer resp.Body.Close() //nolint:errcheck

	// It is read whole, for playing it not to wait for the network.
	audio, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	return m.voicePlayer.Play(attachment.ID, audio)
}

// seekPressMsg is where the waveform of a voice message was pressed, seekMsg where it was dragged to after, and seekReleaseMsg that it was let go of.
type (
	seekPressMsg   time.Duration
	seekMsg        time.Duration
	seekReleaseMsg struct{}
)

func (seekPressMsg) messagesList()   {}
func (seekMsg) messagesList()        {}
func (seekReleaseMsg) messagesList() {}

// seek moves the voice message that is playing or paused to position.
func (m Model) seek(position time.Duration) tview.Cmd {
	return func() tview.Msg {
		if err := m.voicePlayer.Seek(position); err != nil {
			return ui.ErrorDialog("seek voice message", err)
		}

		return nil
	}
}
