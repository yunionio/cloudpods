package seclib2

import "testing"

func TestAes(t *testing.T) {
	secret := "This is a secret for AES!!!"
	key := "This is AES key"

	code, err := encryptAES([]byte(key), []byte(secret))
	if err != nil {
		t.Errorf("encrypt error %s", err)
		return
	}

	secret2, err := decryptAES([]byte(key), code)
	if err != nil {
		t.Errorf("decrypt error %s", err)
		return
	}

	if secret != string(secret2) {
		t.Errorf("aes encrypt/decrypt mismatch! %s != %s", secret, string(secret2))
	}
}
