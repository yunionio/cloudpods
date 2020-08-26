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

package seclib2

import (
	"testing"
)

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
