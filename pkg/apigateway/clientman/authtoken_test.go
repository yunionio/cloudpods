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
	"bytes"
	"compress/flate"
	cryptorand "crypto/rand"
	"encoding/base64"
	"reflect"
	"strings"
	"testing"
	"time"

	"yunion.io/x/onecloud/pkg/apigateway/options"
)

// usePlainCookieKey selects the enable_ssl=false path: no RSA key, AES-GCM cookie.
func usePlainCookieKey(t *testing.T) {
	t.Helper()
	setPrivateKey(nil)
	key := make([]byte, 32)
	if _, err := cryptorand.Read(key); err != nil {
		t.Fatalf("rand: %s", err)
	}
	setCookieKey(key)
	t.Cleanup(SetupTest)
}

func TestEncoeDecode(t *testing.T) {
	SetupTest()
	token := SAuthToken{
		token:      `gAAAAABe-gUMAawOPrP-mA4jY6-b1UPalPJw9WlZJVqHZMtc3IBKUOvHTbKm60YyZQtnVBa3O3QDfS2ss5_Xwi_n0L-jfuUstguLHfDyztAvT_IAKupw8YNK0FvJg25LKC4IR3bmDzCNzTwMO-rEeb4ha2e1vkGOwko9GT1Bn-xN7UM2qeEsm5PiLBg0ZTMuv4Jm5RWIXk2K`,
		verifyTotp: true,
		enableTotp: false,
	}
	enc, err := token.Encode()
	if err != nil {
		t.Fatalf("encode: %s", err)
	}
	dec, err := Decode(enc)
	if err != nil {
		t.Fatalf("decode: %s", err)
	}
	if !reflect.DeepEqual(*dec, token) {
		t.Fatalf("round trip mismatch")
	}
	if strings.Contains(enc, token.token) {
		t.Fatal("keystone token is visible in the sealed cookie")
	}
}

func TestDecodeRejectsTamperedCookie(t *testing.T) {
	usePlainCookieKey(t)
	token := NewAuthToken("keystone-token-of-the-victim", true, true, false)
	enc, err := token.Encode()
	if err != nil {
		t.Fatalf("encode: %s", err)
	}
	raw, err := base64.URLEncoding.DecodeString(enc)
	if err != nil {
		t.Fatalf("decode base64: %s", err)
	}
	raw[len(raw)-1] ^= 0x01
	tampered := base64.URLEncoding.EncodeToString(raw)
	if _, err := Decode(tampered); err == nil {
		t.Fatal("tampered cookie was accepted")
	}

	prev := options.Options
	options.Options = &options.GatewayOptions{EnableTotp: true}
	t.Cleanup(func() { options.Options = prev })

	dec, err := Decode(enc)
	if err != nil {
		t.Fatalf("decode: %s", err)
	}
	if dec.IsTotpVerified() {
		t.Fatal("password login was treated as totp-verified")
	}
	if dec.token != token.token {
		t.Fatalf("token mismatch")
	}
}

func TestDecodeRejectsUnsignedSession(t *testing.T) {
	usePlainCookieKey(t)
	token := NewAuthToken("keystone-token-of-the-victim", true, true, false)
	plain := token.encodeBytes()
	plain[0] = TotpEnable
	if _, err := Decode(legacyCompress(plain)); err == nil {
		t.Fatal("uncompressed session was accepted")
	}
}

func TestDecodeRejectsOtherKey(t *testing.T) {
	usePlainCookieKey(t)
	token := NewAuthToken("keystone-token-of-the-victim", true, true, false)
	enc, err := token.Encode()
	if err != nil {
		t.Fatalf("encode: %s", err)
	}
	setCookieKey(bytes.Repeat([]byte{7}, 32))
	if _, err := Decode(enc); err == nil {
		t.Fatal("cookie sealed with another key was accepted")
	}
}

func TestEncodeRequiresServerKey(t *testing.T) {
	setPrivateKey(nil)
	setCookieKey(nil)
	t.Cleanup(SetupTest)
	token := NewAuthToken("keystone-token-of-the-victim", true, true, false)
	if _, err := token.Encode(); err == nil {
		t.Fatal("encode without a server key succeeded")
	}
}

func TestOtpLockoutIsServerSide(t *testing.T) {
	uid := "totp-lock-user"
	resetOtpLock(uid)
	t.Cleanup(func() { resetOtpLock(uid) })

	for i := 0; i < MAX_OTP_RETRY-1; i++ {
		noteOtpFailure(uid)
	}
	if err := checkOtpLock(uid); err != nil {
		t.Fatalf("locked before max retries: %v", err)
	}
	noteOtpFailure(uid)
	if err := checkOtpLock(uid); err == nil {
		t.Fatal("expected lock after max retries")
	}

	replay := NewAuthToken("keystone-token-2", true, true, false)
	if replay.verifyTotp {
		t.Fatal("fresh login cookie is already verified")
	}
	if err := checkOtpLock(uid); err == nil {
		t.Fatal("fresh login cookie cleared the lock")
	}

	clearOtpLock(uid)
	if err := checkOtpLock(uid); err != nil {
		t.Fatalf("cleared lock still held: %v", err)
	}
}

func TestOtpLockExpires(t *testing.T) {
	uid := "totp-lock-expire"
	resetOtpLock(uid)
	t.Cleanup(func() { resetOtpLock(uid) })

	st := otpFailStateFor(uid)
	st.mu.Lock()
	st.retryCount = MAX_OTP_RETRY
	st.lockUntil = time.Now().Add(-time.Second)
	st.mu.Unlock()

	if err := checkOtpLock(uid); err != nil {
		t.Fatalf("expired lock still held: %v", err)
	}
}

func legacyCompress(in []byte) string {
	buf := new(bytes.Buffer)
	w, err := flate.NewWriter(buf, 9)
	if err != nil {
		panic(err)
	}
	if _, err := w.Write(in); err != nil {
		panic(err)
	}
	if err := w.Close(); err != nil {
		panic(err)
	}
	return base64.URLEncoding.EncodeToString(buf.Bytes())
}
