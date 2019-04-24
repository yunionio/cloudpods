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
	"fmt"

	"github.com/tredoe/osutil/user/crypt/sha512_crypt"

	"yunion.io/x/pkg/util/seclib"
)

func GeneratePassword(passwd string) (string, error) {
	salt := seclib.RandomPassword(8)
	sha512Crypt := sha512_crypt.New()
	return sha512Crypt.Generate([]byte(passwd), []byte(fmt.Sprintf("$6$%s", salt)))
}

func VerifyPassword(passwd string, hash string) error {
	sha512Crypt := sha512_crypt.New()
	return sha512Crypt.Verify(hash, []byte(passwd))
}
