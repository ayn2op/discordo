package login

import "charm.land/bubbles/v2/key"

type keys struct {
	Previous key.Binding
	Next     key.Binding
}

func defaultKeys() keys {
	return keys{
		Previous: key.NewBinding(key.WithKeys("ctrl+p")),
		Next:     key.NewBinding(key.WithKeys("ctrl+n")),
	}
}
