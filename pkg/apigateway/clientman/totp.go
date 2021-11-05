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
	"encoding/base32"

	"yunion.io/x/onecloud/pkg/mcclient"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/identity"
)

const MAX_OTP_RETRY = 5 // totp验证最大重试次数

// 获取用户TOTP credential 密码.
func fetchUserTotpCredSecret(s *mcclient.ClientSession, uid string) (string, error) {
	secret, err := modules.Credentials.GetTotpSecret(s, uid)
	if err != nil {
		return "", err
	}

	return base32.StdEncoding.EncodeToString([]byte(secret)), nil
}
