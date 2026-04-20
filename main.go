package main

import (
	"log"
	"os"

	"github.com/ayn2op/discordo/cmd"
)

func main() {
	if err := cmd.Run(); err != nil {
		log.SetOutput(os.Stderr)
		log.Fatal(err)
	}
}
