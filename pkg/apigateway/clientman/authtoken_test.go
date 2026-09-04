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
	"encoding/base64"
	"testing"

	"yunion.io/x/onecloud/pkg/apigateway/options"
)

var testToken = SAuthToken{
	token:      `gAAAAABe-gUMAawOPrP-mA4jY6-b1UPalPJw9WlZJVqHZMtc3IBKUOvHTbKm60YyZQtnVBa3O3QDfS2ss5_Xwi_n0L-jfuUstguLHfDyztAvT_IAKupw8YNK0FvJg25LKC4IR3bmDzCNzTwMO-rEeb4ha2e1vkGOwko9GT1Bn-xN7UM2qeEsm5PiLBg0ZTMuv4Jm5RWIXk2K`,
	verifyTotp: true,
	enableTotp: false,
}

func TestEncodeDecode(t *testing.T) {
	SetupTest()
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

// a tampered cookie must be rejected
func TestDecodeTampered(t *testing.T) {
	SetupTest()
	enc := testToken.Encode()
	tampered := enc[:len(enc)-8] + "AAAAAAAA"
	if _, err := Decode(tampered); err == nil {
		t.Fatalf("tampered cookie accepted")
	}
	if _, err := Decode("not-a-jwe-token"); err == nil {
		t.Fatalf("garbage cookie accepted")
	}
}

// cookies in the legacy plain (only flate compressed) format must not be
// accepted anymore: their content was forgeable
func TestDecodeLegacyPlainCookieRejected(t *testing.T) {
	SetupTest()
	buf := new(bytes.Buffer)
	compressor, _ := flate.NewWriter(buf, 9)
	compressor.Write(testToken.encodeBytes())
	compressor.Close()
	legacy := base64.URLEncoding.EncodeToString(buf.Bytes())
	if _, err := Decode(legacy); err == nil {
		t.Fatalf("legacy plain cookie accepted")
	}
}

// without an initialized key no cookie may be emitted or decoded
func TestNoKeyRefusesCookie(t *testing.T) {
	saved := privateKey
	privateKey = nil
	defer func() { privateKey = saved }()

	if enc := testToken.Encode(); enc != "" {
		t.Fatalf("cookie emitted without key")
	}
	if _, err := Decode("anything"); err == nil {
		t.Fatalf("decode accepted without key")
	}
}

// InitClient must fail when no key file is configured: cookie protection
// must not silently degrade
func TestInitClientRequiresKeyFile(t *testing.T) {
	if options.Options == nil {
		options.Options = &options.GatewayOptions{}
	}
	saved := options.Options.SslKeyfile
	defer func() { options.Options.SslKeyfile = saved }()

	options.Options.SslKeyfile = ""
	if err := InitClient(); err == nil {
		t.Fatalf("InitClient must fail without ssl_keyfile")
	}
	options.Options.SslKeyfile = "/nonexistent/key.pem"
	if err := InitClient(); err == nil {
		t.Fatalf("InitClient must fail with unreadable ssl_keyfile")
	}
}
