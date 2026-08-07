package token

import (
	"github.com/ayn2op/tview"
)

type TokenMsg string

func tokenCmd(token string) tview.Cmd {
	return func() tview.Msg {
		return TokenMsg(token)
	}
}
