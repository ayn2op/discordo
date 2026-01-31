package main

import (
	"log"

	"github.com/ayn2op/discordo/cmd"
)

func main() {
	if err := cmd.Run(); err != nil {
		log.Fatal(err)
	}
}
