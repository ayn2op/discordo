package ui

import "github.com/ayn2op/tview"

type ModalButton struct {
	Label    string
	Result   tview.Msg
	KeepOpen bool
}

type ModalMsg struct {
	Text    string
	Buttons []ModalButton
}

func ShowModal(text string, buttons ...ModalButton) tview.Cmd {
	return func() tview.Msg { return ModalMsg{Text: text, Buttons: buttons} }
}
