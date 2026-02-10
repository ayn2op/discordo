package key

import charmKey "charm.land/bubbles/v2/key"

func NewBinding(key, desc string) charmKey.Binding {
	return charmKey.NewBinding(charmKey.WithKeys(key), charmKey.WithHelp(key, desc))
}
