package tree

import (
	"fmt"

	"github.com/ayn2op/discordo/internal/ui"
	"github.com/ayn2op/tview"
	"github.com/ayn2op/tview/tree"
)

// YankID copies the ID of node to the clipboard.
func YankID(node *tree.Node) tview.Cmd {
	if node == nil {
		return nil
	}

	id, ok := node.Reference().(fmt.Stringer)
	if !ok {
		return nil
	}
	return ui.WriteClipboard(id.String())
}
