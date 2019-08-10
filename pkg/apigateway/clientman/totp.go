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
	"fmt"
	"time"

	"github.com/pquerna/otp/totp"

	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

const MAX_OTP_RETRY = 5 // totp验证最大重试次数

type ITotp interface {
	UpdateRetryCount()
	IsVerified() bool
	MarkVerified()
	VerifyTotpPasscode(s *mcclient.ClientSession, uid, passcode string) error
}

type ITotpManager interface {
	GetTotp(tid string) ITotp
	SaveTotp(tid string)
}

type STotp struct {
	RetryCount     int       `json:"retry_count"`      // 重试计数器
	LockExpireTime time.Time `json:"lock_expire_time"` // 锁定时间
	Verified       bool      `json:"verified"`         // 验证状态
}

func (self *STotp) UpdateRetryCount() {
	if self.RetryCount < MAX_OTP_RETRY {
		self.RetryCount += 1

		// 锁定
		if self.RetryCount >= MAX_OTP_RETRY {
			self.LockExpireTime = time.Now().Add(30 * time.Second)
		}
	} else {
		// 清零计数器，解除锁定
		if self.LockExpireTime.Before(time.Now()) {
			self.LockExpireTime = time.Now()
			self.RetryCount = 0
		}
	}
}

func (self *STotp) IsVerified() bool {
	return self.Verified
}

func (self *STotp) MarkVerified() {
	self.Verified = true
}

func (self *STotp) VerifyTotpPasscode(s *mcclient.ClientSession, uid, passcode string) error {
	secret, err := fetchUserTotpCredSecret(s, uid)
	if err != nil {
		return fmt.Errorf("fetch totp secrets error: %s", err.Error())
	}

	releaseSeconds := self.LockExpireTime.Unix() - time.Now().Unix()
	if releaseSeconds > 0 {
		return fmt.Errorf("locked, retry after %d seconds", releaseSeconds)
	}

	if totp.Validate(passcode, secret) {
		self.MarkVerified()
		return nil
	}

	self.UpdateRetryCount()
	return fmt.Errorf("invalid passcode")
}

// 获取用户TOTP credential 密码.
func fetchUserTotpCredSecret(s *mcclient.ClientSession, uid string) (string, error) {
	secret, err := modules.Credentials.GetTotpSecret(s, uid)
	if err != nil {
		return "", err
	}

	return base32.StdEncoding.EncodeToString([]byte(secret)), nil
}
