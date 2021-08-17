package util

import "github.com/zalando/go-keyring"

func GetPassword(u string) string {
	p, err := keyring.Get("discordo", u)
	if err != nil {
		return ""
	}

	return p
}

func SetPassword(u string, p string) {
	if err := keyring.Set("discordo", u, p); err != nil {
		panic(err)
	}
}
