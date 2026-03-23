package main

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/ayn2op/discordo/pkg/structdoc"
)

func main() {
	targetFile := os.Getenv("GOFILE")
	targetPackage := os.Getenv("GOPACKAGE")

	out, err := structdoc.Generate(
		targetFile,
		targetPackage,
		".",
		"Config",
		"toml",
	)
	if err != nil {
		panic(err)
	}
	out = "# Configuration\n\n" + strings.TrimLeft(out, "\n")

	root, err := findModuleRoot()
	if err != nil {
		panic(err)
	}
	docDir := filepath.Join(root, "doc")
	if err := os.MkdirAll(docDir, 0o700); err != nil {
		panic(err)
	}
	if err := os.WriteFile(filepath.Join(docDir, "config.md"), []byte(out), 0o600); err != nil {
		panic(err)
	}
}

func findModuleRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		modPath := filepath.Join(dir, "go.mod")
		if _, err := os.Stat(modPath); err == nil {
			return dir, nil
		}
		next := filepath.Dir(dir)
		if next == dir {
			return "", os.ErrNotExist
		}
		dir = next
	}
}
