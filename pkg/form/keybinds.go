package form

import "charm.land/bubbles/v2/key"

type Keybinds struct {
	Next     key.Binding
	Previous key.Binding
	Submit   key.Binding
}

func DefaultKeybinds() Keybinds {
	return Keybinds{
		Previous: key.NewBinding(key.WithKeys("shift+tab"), key.WithHelp("shift+tab", "prev input")),
		Next:     key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "next input")),
		Submit:   key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "submit")),
	}
}
