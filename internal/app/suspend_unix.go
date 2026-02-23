//go:build unix

package app

import (
	"os"
	"os/signal"
	"syscall"
)

func (a *App) suspend() {
	a.inner.Suspend(func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGCONT)
		defer signal.Stop(c)

		_ = syscall.Kill(0, syscall.SIGTSTP)
		<-c
	})
}
