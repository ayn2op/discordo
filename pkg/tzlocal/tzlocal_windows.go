//go:build windows

package tzlocal

import (
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"
)

// getDefaultTimeZone is from the ICU in Windows 10 1903 and later.
var getDefaultTimeZone = windows.NewLazySystemDLL("icu.dll").NewProc("ucal_getDefaultTimeZone")

func system() (string, error) {
	if err := getDefaultTimeZone.Find(); err != nil {
		return "", err
	}

	var name [128]uint16
	var status int32 // UErrorCode
	n, _, _ := getDefaultTimeZone.Call(uintptr(unsafe.Pointer(&name[0])), uintptr(len(name)), uintptr(unsafe.Pointer(&status)))
	if status > 0 {
		return "", fmt.Errorf("ucal_getDefaultTimeZone failed with %d", status)
	}
	return windows.UTF16ToString(name[:int32(n)]), nil
}
