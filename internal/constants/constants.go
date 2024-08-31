package constants

import (
	"os"
	"path/filepath"
)

const Name = "discordo"

const UserAgent = Name + "/0.1 (https://github.com/diamondburned/arikawa, v3)"

const TmpFilePattern = Name + "_*.md"

var ConfigDirPath string

func init() {
	path, err := os.UserConfigDir()
	if err != nil {
		path = "."
	}

	ConfigDirPath = filepath.Join(path, Name)
}
