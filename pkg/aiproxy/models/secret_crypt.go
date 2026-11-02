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

package models

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"

	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"
)

const atRestPrefix = "aescfb64:"

func encryptAtRest(id, plain string) (string, error) {
	id = strings.TrimSpace(id)
	plain = strings.TrimSpace(plain)
	if id == "" {
		return "", errors.Error("empty encrypt id")
	}
	if plain == "" {
		return "", nil
	}
	enc, err := utils.EncryptAESBase64(id, plain)
	if err != nil {
		return "", errors.Wrap(err, "EncryptAESBase64")
	}
	return atRestPrefix + enc, nil
}

// secretPlaintextPlausible reports whether s is usable as an HTTP Authorization
// bearer token / API key (non-empty, no ASCII control characters).
//
// AES-CFB decryption has no integrity check: DescryptAESBase64 can "succeed" on
// hex or base64-looking plaintext and return binary garbage. Callers must reject
// that garbage so sealSecret does not permanently corrupt user-entered secrets.
func secretPlaintextPlausible(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c < 0x20 || c == 0x7f {
			return false
		}
	}
	return true
}

func decryptAtRest(id, stored string) string {
	stored = strings.TrimSpace(stored)
	if stored == "" {
		return ""
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return stored
	}
	if strings.HasPrefix(stored, atRestPrefix) {
		plain, err := utils.DescryptAESBase64(id, strings.TrimPrefix(stored, atRestPrefix))
		if err != nil || !secretPlaintextPlausible(plain) {
			return ""
		}
		return plain
	}
	// Unprefixed rows may still be legacy ciphertext (pre-aescfb64 marker) or
	// raw plaintext. Only accept a CFB decrypt when the result looks like a key;
	// otherwise keep the stored value as plaintext (hex API keys, etc.).
	plain, err := utils.DescryptAESBase64(id, stored)
	if err != nil || !secretPlaintextPlausible(plain) {
		return stored
	}
	return plain
}

func secretLooksEncrypted(stored string) bool {
	return strings.HasPrefix(strings.TrimSpace(stored), atRestPrefix)
}

func virtualKeyDigest(plain string) string {
	plain = strings.TrimSpace(plain)
	if plain == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(plain))
	return hex.EncodeToString(sum[:])
}

func queryUnprefixedSecret(q *sqlchemy.SQuery, field string) *sqlchemy.SQuery {
	return q.IsNotEmpty(field).Filter(sqlchemy.NOT(sqlchemy.Startswith(q.Field(field), atRestPrefix)))
}
