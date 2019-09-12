package utils

import (
	"testing"

	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

func TestKey(t *testing.T) {
	t.Run("parse", func(t *testing.T) {
		keyStr := "WJYVsrTtAae1QS9YzefV4OmVM6mkJglR+GEgxQpTs2g="
		pubkeyStr := "mOX0S5AuRqd8lQZWcqTlzOS+veo404gE7NyV4u3xVkg="
		pubkeyX, err := wgtypes.ParseKey(pubkeyStr)
		if err != nil {
			t.Fatalf("parse public key failed: %v", err)
		}

		key, err := wgtypes.ParseKey(keyStr)
		if err != nil {
			t.Fatalf("parse private key failed: %v", err)
		}
		pubkey := key.PublicKey()
		if pubkey != pubkeyX {
			t.Errorf("derived public key does not match expected.\ngot %#v\nwant %#v", pubkey, pubkeyX)
		}
	})
	t.Run("generate", func(t *testing.T) {
		key, err := wgtypes.GeneratePrivateKey()
		if err != nil {
			t.Fatalf("generate private key failed: %v", err)
		}
		pubkey := key.PublicKey()
		t.Logf("private key: %s", key)
		t.Logf(" public key: %s", pubkey)
	})
}
