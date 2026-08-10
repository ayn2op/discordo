//go:build !unix

package root

import "github.com/ayn2op/tview"

func suspend() tview.Cmd { return nil }
