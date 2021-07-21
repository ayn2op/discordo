package util

import (
	"github.com/zalando/go-keyring"
)

const Service string = "discordo"

func GetPassword(user string) (password string) {
	password, err := keyring.Get(Service, user)
	if err == keyring.ErrNotFound {
		return ""
	}

	return
}

func SetPassword(user string, password string) {
	if err := keyring.Set(Service, user, password); err != nil {
		panic(err)
	}
}
