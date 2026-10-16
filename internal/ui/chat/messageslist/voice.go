package messageslist

import (
	"fmt"
	"slices"
	"time"

	"github.com/ayn2op/arikawa/v3/discord"
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

// voiceMessageView returns the player of the voice message a: whether it is playing, its waveform with what was played of it, and how long it is.
func (m Model) voiceMessageView(a discord.Attachment) tview.Widget {
	duration := time.Duration(a.DurationSecs * float64(time.Second))
	symbol, label := symbolPlay, formatDuration(duration)

	status := m.voiceStatus
	loaded := status.ID == a.ID
	if loaded {
		label = formatDuration(status.Position) + positionSeparator + label
	}

	if loaded && status.Playing {
		symbol = symbolPause
	}

	rail := waveformRail(a.Waveform)
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

	width, _ := text.New(symbol).Size()
	play := button.New().Label(symbol).Width(width).OnClick(playMsg(a))

	return row.New(play, waveform, text.New(label)).Spacing(1).Height(layout.Fixed(1))
}
