package tea

import tcell "github.com/gdamore/tcell/v3"

type (
	Msg any
	Cmd func() Msg
)

type (
	KeyMsg    = tcell.EventKey
	MouseMsg  = tcell.EventMouse
	ResizeMsg = tcell.EventResize
	PasteMsg  = tcell.EventPaste
	FocusMsg  = tcell.EventFocus
)

type quitMsg struct{}

func Quit() Msg {
	return quitMsg{}
}

type batchMsg []Cmd
type sequenceMsg []Cmd

func Batch(cmds ...Cmd) Cmd {
	filtered := compactCmds(cmds)
	switch len(filtered) {
	case 0:
		return nil
	case 1:
		return filtered[0]
	default:
		return func() Msg { return batchMsg(filtered) }
	}
}

func Sequence(cmds ...Cmd) Cmd {
	filtered := compactCmds(cmds)
	switch len(filtered) {
	case 0:
		return nil
	case 1:
		return filtered[0]
	default:
		return func() Msg { return sequenceMsg(filtered) }
	}
}

func compactCmds(cmds []Cmd) []Cmd {
	filtered := make([]Cmd, 0, len(cmds))
	for _, cmd := range cmds {
		if cmd != nil {
			filtered = append(filtered, cmd)
		}
	}
	return filtered
}
