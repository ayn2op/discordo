package util

import "github.com/zalando/go-keyring"

// GetPassword retrieves the password in keyring for provided user.
func GetPassword(u string) string {
	p, err := keyring.Get("discordo", u)
	if err != nil {
		return ""
	}

	return p
}

// SetPassword sets the password in keyring for provided user.
func SetPassword(u string, p string) {
	if err := keyring.Set("discordo", u, p); err != nil {
		panic(err)
	}
}
