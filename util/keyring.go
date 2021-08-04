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

func GetItem(kr keyring.Keyring, key string) string {
	item, err := kr.Get(key)
	if err != nil {
		return ""
	}

	return string(item.Data)
}

func SetItem(kr keyring.Keyring, key string, data string) {
	if err := kr.Set(keyring.Item{Key: key, Data: []byte(data)}); err != nil {
		panic(err)
	}
}
