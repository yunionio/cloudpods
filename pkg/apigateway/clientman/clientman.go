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
	"os"
	"path/filepath"

	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/apigateway/options"
	"yunion.io/x/onecloud/pkg/util/seclib2"
)

const sessionKeySize = 32

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
	} else {
		key, err := loadOrCreateSessionKey(options.Options.SessionKeyFile)
		if err != nil {
			return errors.Wrap(err, "loadOrCreateSessionKey")
		}
		setSessionKey(key)
	}

	return nil
}

// loadOrCreateSessionKey returns the persistent key protecting session
// cookies when SSL is not enabled. The key is generated on first start and
// must be shared across apigateway instances.
func loadOrCreateSessionKey(path string) ([]byte, error) {
	if len(path) == 0 {
		return nil, errors.Error("empty session key file path")
	}
	keyBytes, err := os.ReadFile(path)
	if err == nil {
		if len(keyBytes) != sessionKeySize {
			return nil, errors.Errorf("invalid session key file %s", path)
		}
		return keyBytes, nil
	}
	if !os.IsNotExist(err) {
		return nil, errors.Wrapf(err, "os.ReadFile %s", path)
	}
	key := make([]byte, sessionKeySize)
	if _, err := rand.Read(key); err != nil {
		return nil, errors.Wrap(err, "rand.Read")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return nil, errors.Wrap(err, "MkdirAll")
	}
	if err := os.WriteFile(path, key, 0600); err != nil {
		return nil, errors.Wrapf(err, "os.WriteFile %s", path)
	}
	return key, nil
}

func SetupTest() {
	key, _ := rsa.GenerateKey(rand.Reader, 2048)
	setPrivateKey(key)
}
