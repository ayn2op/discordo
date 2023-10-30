package main

import (
	"flag"
	"log"

	"github.com/ayn2op/discordo/cmd"
	"github.com/ayn2op/discordo/internal/constants"
	"github.com/zalando/go-keyring"
)

func main() {
	t, _ := keyring.Get(constants.Name, "token")
	token := flag.String("token", t, "The authentication token.")
	flag.Parse()

	if err := cmd.Run(*token); err != nil {
		log.Fatal(err)
	}
}
