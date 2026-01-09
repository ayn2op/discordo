//go:build !unix || windows

package media

func getCellSizeFromTerminal() (int, int, bool) {
	return 0, 0, false
}
