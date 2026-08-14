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
		if err != nil {
			return ""
		}
		return plain
	}
	plain, err := utils.DescryptAESBase64(id, stored)
	if err != nil {
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
