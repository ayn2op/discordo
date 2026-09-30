package token

import (
	"github.com/ayn2op/tview"
	"github.com/ayn2op/tview/textinput"
)

type TokenMsg string

type (
	tokenMsg       textinput.Change
	focusButtonMsg struct{}
	submitMsg      struct{}
)

func submitToken(token string) tview.Cmd {
	return func() tview.Msg {
		return TokenMsg(token)
	}
}
