package tree

import "charm.land/bubbles/v2/key"

type Keybinds struct {
	Previous key.Binding
	Next     key.Binding
	First    key.Binding
	Last     key.Binding
}

func DefaultKeybinds() Keybinds {
	return Keybinds{
		Previous: key.NewBinding(key.WithKeys("k"), key.WithHelp("k", "prev")),
		Next:     key.NewBinding(key.WithKeys("j"), key.WithHelp("j", "next")),
		First:    key.NewBinding(key.WithKeys("g"), key.WithHelp("g", "first")),
		Last:     key.NewBinding(key.WithKeys("G"), key.WithHelp("G", "last")),
	}
}
