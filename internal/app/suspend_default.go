//go:build !unix

package app

func (a *App) suspend() {}
