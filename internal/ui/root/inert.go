package root

import "github.com/ayn2op/tview"

// inert draws child but gives it no input, like HTML's inert attribute, for what is behind the dialog.
type inert struct {
	child tview.Element
}

var _ tview.Element = inert{}

func (i inert) Draw(screen tview.Screen, area tview.Rectangle) {
	i.child.Draw(screen, area)
}

func (inert) Handle(msg tview.Msg, area tview.Rectangle) tview.Msg {
	return msg
}
