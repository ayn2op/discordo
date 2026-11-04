package token

import (
	"github.com/ayn2op/tview"
	"github.com/ayn2op/tview/cmd"
)

type TokenMsg string

// tokenMsg is the token and cursor that typing in the input left.
type tokenMsg struct {
	text   string
	cursor int
}

type (
	focusButtonMsg struct{}
	hoverMsg       bool
	submitMsg      struct{}
)

func submitToken(token string) tview.Cmd {
	return cmd.Task(func() tview.Msg {
		return TokenMsg(token)
	})
}
