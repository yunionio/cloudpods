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

	"github.com/pkg/errors"

	"yunion.io/x/onecloud/pkg/apigateway/options"
	"yunion.io/x/onecloud/pkg/util/seclib2"
)

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
	}

	return nil
}

func SetupTest() {
	key, _ := rsa.GenerateKey(rand.Reader, 2048)
	setPrivateKey(key)
}
