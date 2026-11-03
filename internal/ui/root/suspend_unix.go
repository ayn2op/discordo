//go:build unix

package root

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/ayn2op/tview"
	"github.com/ayn2op/tview/cmd"
)

func suspend() tview.Cmd {
	return cmd.Suspend(func() tview.Msg {
		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGCONT)
		defer signal.Stop(c)

		_ = syscall.Kill(0, syscall.SIGTSTP)
		<-c
		return nil
	})
}
