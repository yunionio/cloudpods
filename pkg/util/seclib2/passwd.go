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
	"github.com/tredoe/osutil/user/crypt/sha512_crypt"
	"golang.org/x/crypto/bcrypt"

	"yunion.io/x/pkg/util/seclib"
)

func GeneratePassword(passwd string) (string, error) {
	return seclib.GeneratePassword(passwd)
}

func VerifyPassword(passwd string, hash string) error {
	sha512Crypt := sha512_crypt.New()
	return sha512Crypt.Verify(hash, []byte(passwd))
}

func BcryptPassword(passwd string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(passwd), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), err
}

func BcryptVerifyPassword(passwd string, hash string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(passwd))
}
