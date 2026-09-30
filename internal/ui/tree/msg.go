package tree

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/ayn2op/tview"
	"github.com/ayn2op/tview/tree"
	"golang.design/x/clipboard"
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
	return func() tview.Msg {
		if _, err := clipboard.Write(context.Background(), clipboard.FmtText, []byte(id.String())); err != nil {
			slog.Error("failed to write to clipboard", "err", err)
		}
		return nil
	}
}
