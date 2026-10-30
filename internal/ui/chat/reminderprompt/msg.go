package reminderprompt

import "github.com/ayn2op/tview"

// SubmittedMsg carries the command for what was typed.
type SubmittedMsg struct {
	Action tview.Cmd
}

type CancelMsg struct{}

// inputMsg is the text and cursor that typing left.
type inputMsg struct {
	text   string
	cursor int
}

type submitMsg struct{}
