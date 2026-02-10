package tabs

import "charm.land/bubbles/v2/key"

type Keybinds struct {
	Previous key.Binding
	Next     key.Binding
}

func DefaultKeybinds() Keybinds {
	return Keybinds{
		Previous: key.NewBinding(key.WithKeys("ctrl+h"), key.WithHelp("ctrl+h", "prev tab")),
		Next:     key.NewBinding(key.WithKeys("ctrl+l"), key.WithHelp("ctrl+l", "next tab")),
	}
}
