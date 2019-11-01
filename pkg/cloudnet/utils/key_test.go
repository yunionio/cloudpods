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
