package main

import (
	"flag"
	"log"

	"github.com/ayn2op/discordo/cmd"
	"github.com/ayn2op/discordo/internal/constants"
	"github.com/zalando/go-keyring"
)

func main() {
	t, err := keyring.Get(constants.Name, "token")
	if err != nil {
		log.Println("token not found in keyring:", err)
	}

	token := flag.String("token", t, "The authentication token.")
	flag.Parse()

	if err := cmd.Run(*token); err != nil {
		log.Fatal(err)
	}
}
