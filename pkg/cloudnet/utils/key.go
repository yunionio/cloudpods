package utils

import (
	"fmt"

	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

func MustNewKey() wgtypes.Key {
	key, err := wgtypes.GeneratePrivateKey()
	if err != nil {
		panic(fmt.Errorf("new key: %v", err))
	}
	return key
}

func MustNewKeyString() string {
	key := MustNewKey()
	return key.String()
}

func MustParseKeyString(k string) wgtypes.Key {
	key, err := wgtypes.ParseKey(k)
	if err != nil {
		panic(fmt.Errorf("invalid key: %v", err))
	}
	return key
}
