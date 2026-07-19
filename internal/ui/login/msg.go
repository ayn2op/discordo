package login

import (
	"github.com/ayn2op/tview"
	"golang.design/x/clipboard"
)

type errMsg struct {
	err error
}

func setClipboard(content string) tview.Cmd {
	return func() tview.Msg {
		_ = clipboard.Write(clipboard.FmtText, []byte(content))
		return nil
	}
}
