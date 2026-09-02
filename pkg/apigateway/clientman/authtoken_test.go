// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package clientman

import (
	cryptorand "crypto/rand"
	"encoding/base64"
	"testing"
)

var testToken = SAuthToken{
	token:      `gAAAAABe-gUMAawOPrP-mA4jY6-b1UPalPJw9WlZJVqHZMtc3IBKUOvHTbKm60YyZQtnVBa3O3QDfS2ss5_Xwi_n0L-jfuUstguLHfDyztAvT_IAKupw8YNK0FvJg25LKC4IR3bmDzCNzTwMO-rEeb4ha2e1vkGOwko9GT1Bn-xN7UM2qeEsm5PiLBg0ZTMuv4Jm5RWIXk2K`,
	verifyTotp: true,
	enableTotp: false,
}

func TestEncodeDecodeRsa(t *testing.T) {
	SetupTest()
	encEt := EncryptString(testToken.encodeBytes())
	decBytes, err := DecryptString(encEt)
	if err != nil {
		t.Fatalf("decryptString fail %s", err)
	}
	token2, err := decodeBytes(decBytes)
	if err != nil {
		t.Fatalf("decodeBytes fail %s", err)
	}
	if *token2 != testToken {
		t.Fatalf("token2 != token")
	}
}

func setupSessionKey(t *testing.T) {
	privateKey = nil
	key := make([]byte, sessionKeySize)
	if _, err := cryptorand.Read(key); err != nil {
		t.Fatalf("rand read: %v", err)
	}
	setSessionKey(key)
}

func TestEncodeDecodeSessionKey(t *testing.T) {
	setupSessionKey(t)
	enc := testToken.Encode()
	if enc == "" {
		t.Fatalf("empty encoded cookie")
	}
	token2, err := Decode(enc)
	if err != nil {
		t.Fatalf("Decode fail %s", err)
	}
	if *token2 != testToken {
		t.Fatalf("token2 != token")
	}
}

func TestDecodeTamperedSessionCookie(t *testing.T) {
	setupSessionKey(t)
	enc := testToken.Encode()
	raw, err := base64.URLEncoding.DecodeString(enc)
	if err != nil {
		t.Fatalf("base64 decode: %v", err)
	}
	// flip a bit in the ciphertext (GCM auth tag is at the end)
	raw[len(raw)-1] ^= 0x01
	tampered := base64.URLEncoding.EncodeToString(raw)
	if _, err := Decode(tampered); err == nil {
		t.Fatalf("tampered cookie accepted")
	}
}

func TestDecodeForgedPlaintextCookie(t *testing.T) {
	setupSessionKey(t)
	// old-style plain (only compressed) cookies must no longer be accepted
	plain := base64.URLEncoding.EncodeToString(testToken.encodeBytes())
	if _, err := Decode(plain); err == nil {
		t.Fatalf("plaintext cookie accepted")
	}
}
