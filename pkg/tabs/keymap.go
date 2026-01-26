package tabs

import "charm.land/bubbles/v2/key"

type Keybinds struct {
	Previous key.Binding
	Next     key.Binding
}

func DefaultKeybinds() Keybinds {
	return Keybinds{
		Previous: key.NewBinding(key.WithKeys("ctrl+p")),
		Next:     key.NewBinding(key.WithKeys("ctrl+n")),
	}
}
