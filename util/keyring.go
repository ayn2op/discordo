package util

import (
	"github.com/99designs/keyring"
)

const ServiceName string = "discordo"

func OpenKeyringBackend() keyring.Keyring {
	kr, err := keyring.Open(keyring.Config{
		ServiceName: ServiceName,
	})
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
