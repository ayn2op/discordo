package token

import (
	"github.com/ayn2op/tview"
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
	return func() tview.Msg {
		return TokenMsg(token)
	}
}
