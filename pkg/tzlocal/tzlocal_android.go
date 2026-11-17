package tzlocal

import (
	"errors"
	"os/exec"
	"strings"
)

func system() (string, error) {
	output, err := exec.Command("/system/bin/getprop", "persist.sys.timezone").Output()
	if err != nil {
		return "", err
	}

	name := strings.TrimSpace(string(output))
	if name == "" {
		return "", errors.New("persist.sys.timezone is empty")
	}
	return name, nil
}
