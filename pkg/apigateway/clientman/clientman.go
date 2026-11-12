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
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"os"
	"strings"

	"github.com/pkg/errors"

	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/apigateway/options"
	"yunion.io/x/onecloud/pkg/util/seclib2"
)

// cookieKeyContext separates this use from other hashes of the same secret.
const cookieKeyContext = "onecloud-auth-cookie-key-v1\x00"

func InitClient() error {
	if options.Options.EnableSsl {
		privData, err := os.ReadFile(options.Options.SslKeyfile)
		if err != nil {
			return errors.Wrapf(err, "os.ReadFile %s", options.Options.SslKeyfile)
		}
		privateKey, err := seclib2.DecodePrivateKey(privData)
		if err != nil {
			return errors.Wrap(err, "decodePrivateKey")
		}
		setPrivateKey(privateKey)
		return nil
	}
	return initCookieKey()
}

// initCookieKey runs only when TLS is off. With enable_ssl the session is
// still encrypted with the certificate private key.
func initCookieKey() error {
	if secret := strings.TrimSpace(options.Options.AuthCookieKey); secret != "" {
		sum := sha256.Sum256(append([]byte(cookieKeyContext), secret...))
		setCookieKey(sum[:])
		log.Infof("auth cookie key loaded from auth-cookie-key")
		return nil
	}
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return errors.Wrap(err, "generate auth cookie key")
	}
	setCookieKey(key)
	log.Warningf("auth cookie key is ephemeral; sessions end when apigateway restarts and are not valid on other replicas. Set auth-cookie-key to a shared secret.")
	return nil
}

func SetupTest() {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		panic(err)
	}
	setPrivateKey(key)
	setCookieKey(nil)
}
