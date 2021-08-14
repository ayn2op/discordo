package util

import (
	"github.com/99designs/keyring"
)

func OpenKeyringBackend() keyring.Keyring {
	kr, err := keyring.Open(keyring.Config{})
	if err != nil {
		panic(err)
	}

	return kr
}

func GetItem(kr keyring.Keyring, k string) string {
	item, err := kr.Get(k)
	if err != nil {
		return ""
	}

	return string(item.Data)
}

func SetItem(kr keyring.Keyring, k string, d string) {
	if err := kr.Set(keyring.Item{Key: k, Data: []byte(d)}); err != nil {
		panic(err)
	}
}
