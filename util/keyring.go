package util

import (
	"github.com/99designs/keyring"
)

func OpenKeyringBackend() (kr keyring.Keyring) {
	kr, err := keyring.Open(keyring.Config{})
	if err != nil {
		panic(err)
	}

	return
}

func GetItem(kr keyring.Keyring, k string) string {
	i, err := kr.Get(k)
	if err != nil {
		return ""
	}

	return string(i.Data)
}

func SetItem(kr keyring.Keyring, k string, d string) {
	if err := kr.Set(keyring.Item{Key: k, Data: []byte(d)}); err != nil {
		panic(err)
	}
}
