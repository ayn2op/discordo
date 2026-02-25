//go:build unix

package root

import (
	"os"
	"os/signal"
	"syscall"
)

func (v *View) suspend() {
	v.app.Suspend(func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGCONT)
		defer signal.Stop(c)

		_ = syscall.Kill(0, syscall.SIGTSTP)
		<-c
	})
}
