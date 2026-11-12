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
	"sync"
	"time"

	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/identity"
)

const MAX_OTP_RETRY = 5 // totp验证最大重试次数

// otpLockDuration is how long a user stays locked after MAX_OTP_RETRY failures.
// The counter lives in this process, keyed by user id, so a replayed login
// cookie cannot reset it.
var otpLockDuration = 30 * time.Second

type otpFailState struct {
	mu         sync.Mutex
	retryCount int
	lockUntil  time.Time
}

var otpFails sync.Map // userId -> *otpFailState

func otpFailStateFor(uid string) *otpFailState {
	if uid == "" {
		uid = "-"
	}
	v, _ := otpFails.LoadOrStore(uid, &otpFailState{})
	return v.(*otpFailState)
}

func noteOtpFailure(uid string) {
	st := otpFailStateFor(uid)
	st.mu.Lock()
	defer st.mu.Unlock()
	now := time.Now()
	if !st.lockUntil.IsZero() && !now.Before(st.lockUntil) {
		st.lockUntil = time.Time{}
		st.retryCount = 0
	}
	st.retryCount++
	if st.retryCount >= MAX_OTP_RETRY {
		st.lockUntil = now.Add(otpLockDuration)
	}
}

func checkOtpLock(uid string) error {
	if uid == "" {
		uid = "-"
	}
	v, ok := otpFails.Load(uid)
	if !ok {
		return nil
	}
	st := v.(*otpFailState)
	st.mu.Lock()
	defer st.mu.Unlock()
	now := time.Now()
	if !st.lockUntil.IsZero() && now.Before(st.lockUntil) {
		secs := int(st.lockUntil.Sub(now).Seconds())
		if secs < 1 {
			secs = 1
		}
		return errors.Wrapf(httperrors.ErrResourceBusy, "locked, retry after %d seconds", secs)
	}
	return nil
}

func clearOtpLock(uid string) {
	if uid == "" {
		uid = "-"
	}
	otpFails.Delete(uid)
}

func resetOtpLock(uid string) {
	clearOtpLock(uid)
}

// 获取用户TOTP credential 密码.
func fetchUserTotpCredSecret(s *mcclient.ClientSession, uid string) (string, error) {
	secret, err := modules.Credentials.GetTotpSecret(s, uid)
	if err != nil {
		return "", err
	}

	return base32.StdEncoding.EncodeToString([]byte(secret)), nil
}
