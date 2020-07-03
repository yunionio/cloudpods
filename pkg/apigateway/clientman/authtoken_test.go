package clientman

import (
	"crypto/rand"
	"crypto/rsa"
	"reflect"
	"testing"
)

func TestEncoeDecode(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	token := SAuthToken{
		token:      `gAAAAABe-gUMAawOPrP-mA4jY6-b1UPalPJw9WlZJVqHZMtc3IBKUOvHTbKm60YyZQtnVBa3O3QDfS2ss5_Xwi_n0L-jfuUstguLHfDyztAvT_IAKupw8YNK0FvJg25LKC4IR3bmDzCNzTwMO-rEeb4ha2e1vkGOwko9GT1Bn-xN7UM2qeEsm5PiLBg0ZTMuv4Jm5RWIXk2K`,
		verifyTotp: true,
		enableTotp: false,
	}
	setPrivateKey(key)
	et := token.encodeBytes()
	plainEt := compressString(et)
	encEt := encryptString(et)
	t.Logf("origin token: %s", token.token)
	t.Logf("plain token: %s (%d)", plainEt, len(plainEt))
	t.Logf("encrypt token: %s (%d)", encEt, len(encEt))

	decBytes, err := decompressString(plainEt)
	if err != nil {
		t.Fatalf("decompressString fail %s", err)
	}
	decBytes2, err := decryptString(encEt)
	if err != nil {
		t.Fatalf("decryptString fail %s", err)
	}

	if !reflect.DeepEqual(decBytes, decBytes2) {
		t.Fatalf("decompress and decrypt bytes not equal!")
	}

	token2, err := decodeBytes(decBytes)
	if err != nil {
		t.Fatalf("decodeBytes fail %s", err)
	}
	if *token2 != token {
		t.Fatalf("token2 != token")
	}
}
